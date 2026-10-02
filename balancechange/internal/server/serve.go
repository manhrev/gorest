package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/manhrev/gorest/balancechange/config"
	"github.com/manhrev/gorest/balancechange/internal/gemini"
	txrepo "github.com/manhrev/gorest/balancechange/internal/repository/transaction"
	txservice "github.com/manhrev/gorest/balancechange/internal/service/transaction"
	applog "github.com/manhrev/mkit/log"
	"github.com/manhrev/mkit/middleware"
	"github.com/manhrev/mkit/postgres"
	"github.com/manhrev/mkit/tracing"
)

// Run sets up every dependency and serves until ctx is canceled (or
// ListenAndServe fails outright). Returns the first fatal error instead of
// exiting the process itself, so callers' own deferred cleanup still runs.
func Run(ctx context.Context) error {
	cfg := config.Load()
	if cfg.APIKey == "" {
		return errors.New("API_KEY is required")
	}
	if cfg.Gemini.APIKey == "" {
		return errors.New("GEMINI_API_KEY is required")
	}

	tracingSvc, err := tracing.NewService(ctx, &cfg.App)
	if err != nil {
		return fmt.Errorf("init tracing: %w", err)
	}
	defer func() {
		if err := tracingSvc.Stop(context.Background()); err != nil {
			applog.Bootstrap().Error("stop tracing", "error", err)
		}
	}()

	// Fan out to console (always) + OTEL log export (only when the log
	// signal is enabled; tracingSvc.Handler() is nil otherwise, and
	// NewLogger drops nil handlers).
	logger := applog.NewLogger(&cfg.App, tracingSvc.Handler())

	pgPool, err := postgres.New(ctx, &cfg.App, logger)
	if err != nil {
		return fmt.Errorf("init postgres: %w", err)
	}
	defer pgPool.Close() // blocks until in-flight connections are returned, see pool.Close() doc

	router := http.NewServeMux()
	humaConfig := huma.DefaultConfig("Balance Change API", "1.0.0")
	humaConfig.Components.SecuritySchemes = map[string]*huma.SecurityScheme{
		apiKeySchemeName: {Type: "apiKey", In: "header", Name: apiKeyHeader},
	}
	api := humago.New(router, humaConfig)
	api.UseMiddleware(apiKeyAuth(api, cfg.APIKey))
	registerSwaggerRoute(router)

	srv := NewServer(txservice.New(txrepo.New(pgPool), gemini.New(cfg.Gemini.APIKey, cfg.Gemini.Model)))
	srv.registerTransactionRoutes(api, "/api/v1/transaction")

	handler := middleware.CORS(cfg.AllowedOrigins)(
		middleware.Metadata(cfg.App.Version)(
			middleware.Logger(logger)(
				middleware.Recoverer(router),
			),
		),
	)

	addr := cfg.HTTP.Host + ":" + cfg.HTTP.Port
	// otelhttp wraps the whole chain (CORS→Metadata→Logger→Recoverer→router)
	// in a root span per request, named "METHOD /path" instead of otelhttp's
	// default (the operation arg verbatim — every request would otherwise
	// share one span name). Safe unconditionally: when tracing is disabled,
	// no TracerProvider is ever registered, so this falls back to the global
	// no-op provider (negligible overhead).
	httpSrv := &http.Server{
		Addr: addr,
		Handler: otelhttp.NewHandler(handler, "",
			otelhttp.WithSpanNameFormatter(func(_ string, r *http.Request) string {
				return r.Method + " " + r.URL.Path
			}),
		),
	}

	// On SIGINT/SIGTERM (ctx canceled by the caller): stop taking new
	// requests, let in-flight ones finish (bounded by the timeout), then
	// unblock ListenAndServe below so the deferred cleanup above runs.
	go func() {
		<-ctx.Done()
		logger.Info("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()
		if err := httpSrv.Shutdown(shutdownCtx); err != nil {
			logger.Error("http shutdown", "error", err)
		}
	}()

	logger.Info("starting server", "addr", "http://"+addr, "docs", "/docs", "swagger", "/swagger", "spec", "/openapi.json")
	if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("server stopped: %w", err)
	}

	return nil
}

// registerSwaggerRoute serves Swagger UI from the CDN, pointed at the
// auto-generated spec. (humago already registers /docs for its default
// Stoplight Elements UI.)
func registerSwaggerRoute(router *http.ServeMux) {
	router.HandleFunc("GET /swagger", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<!doctype html>
			<html>
			<head>
				<title>API Docs</title>
				<link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist/swagger-ui.css">
			</head>
			<body>
				<div id="swagger-ui"></div>
				<script src="https://unpkg.com/swagger-ui-dist/swagger-ui-bundle.js"></script>
				<script>
					SwaggerUIBundle({url: '/openapi.json', dom_id: '#swagger-ui'})
				</script>
			</body>
			</html>`))
	})
}
