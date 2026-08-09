// Package config builds the app's Config from environment variables. The
// generic app types live in pkg/config; this package adds server-specific
// fields on top and does the env -> struct wiring, kept separate so
// pkg/config stays pure data.
package config

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"

	pkgconfig "github.com/manhrev/gorest/pkg/config"
)

// Config is pkgconfig.App plus fields specific to this server binary
// (not generic enough to belong in pkg/config).
type Config struct {
	pkgconfig.App

	ShutdownTimeout time.Duration
	AllowedOrigins  []string
}

// Load builds a *Config from environment variables, with sane defaults for
// local development.
func Load() *Config {
	// silently ignored if .env absent — fine for optional local
	// override; real env vars still take precedence since Load already
	// reads them, and godotenv.Load doesn't overwrite ones already set.
	_ = godotenv.Load()

	return &Config{
		App: pkgconfig.App{
			Version: envOr("APP_VERSION", "dev"),
			HTTP: &pkgconfig.HTTP{
				Host: envOr("HTTP_HOST", "localhost"),
				Port: envOr("HTTP_PORT", "8080"),
			},
			Log: &pkgconfig.Log{Level: os.Getenv("LOG_LEVEL")},
			Tracing: pkgconfig.Tracing{
				ServiceName:   "app",
				Enabled:       envBoolOr("TRACING_ENABLED", false),
				CollectorHost: envOr("TRACING_COLLECTOR_HOST", "localhost"),
				CollectorPort: 4317,
				Secure:        envBoolOr("TRACING_SECURE", false),
				Trace:         envBoolOr("TRACING_TRACE", true),
				Metric:        envBoolOr("TRACING_METRIC", false),
				Log:           envBoolOr("TRACING_LOG", false),
			},
			Postgres: &pkgconfig.Postgres{
				ConnectionParams: &pkgconfig.PostgresConnectionParams{
					Host:     envOr("POSTGRES_HOST", "localhost"),
					Port:     envIntOr("POSTGRES_PORT", 5432),
					User:     envOr("POSTGRES_USER", "postgres"),
					Password: envOr("POSTGRES_PASSWORD", "postgres"),
					DBName:   envOr("POSTGRES_DB", "app"),
					SSLMode:  envOr("POSTGRES_SSLMODE", "disable"),
				},
				IsMigrateSchema: envBoolOr("POSTGRES_MIGRATE", false),
				MaxOpenConn:     10,
				MaxIdleConn:     5,
			},
		},
		ShutdownTimeout: envDurationOr("SHUTDOWN_TIMEOUT", 5*time.Second),
		AllowedOrigins:  envSliceOr("CORS_ALLOWED_ORIGINS", []string{"*"}),
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envIntOr(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func envBoolOr(key string, def bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return def
}

func envSliceOr(key string, def []string) []string {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	parts := strings.Split(v, ",")
	for i, p := range parts {
		parts[i] = strings.TrimSpace(p)
	}
	return parts
}

func envDurationOr(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}
