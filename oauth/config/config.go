// Package config builds the app's Config from environment variables. The
// generic app types live in common/config; this package adds server-specific
// fields on top and does the env -> struct wiring, kept separate so
// common/config stays pure data.
package config

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"

	pkgconfig "github.com/manhrev/gorest/common/config"
)

// Config is pkgconfig.App plus fields specific to this server binary
// (not generic enough to belong in common/config).
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
		Version: envOr("APP_VERSION", "dev"),
		HTTP: &pkgconfig.HTTP{
			Host: envOr("HTTP_HOST", "localhost"),
			// 8081, not huma-bob's 8080 — this demo runs alongside it.
			Port: envOr("HTTP_PORT", "8081"),
		},
		Log: &pkgconfig.Log{Level: os.Getenv("LOG_LEVEL")},
		Tracing: pkgconfig.Tracing{
			ServiceName:   "oauth",
			Enabled:       envBoolOr("TRACING_ENABLED", false),
			CollectorHost: envOr("TRACING_COLLECTOR_HOST", "localhost"),
			CollectorPort: 4317,
			Secure:        envBoolOr("TRACING_SECURE", false),
			Trace:         envBoolOr("TRACING_TRACE", true),
			Metric:        envBoolOr("TRACING_METRIC", false),
			Log:           envBoolOr("TRACING_LOG", false),
		},
		JWT: &pkgconfig.JWT{
			PrivateKeyFile:       envOr("JWT_PRIVATE_KEY_FILE", "../common/jwtmanager/testdata/priv.pem"),
			PublicKeyFile:        envOr("JWT_PUBLIC_KEY_FILE", "../common/jwtmanager/testdata/pub.pem"),
			AccessTokenDuration:  envDurationOr("JWT_ACCESS_TOKEN_DURATION", 15*time.Minute),
			RefreshTokenDuration: envDurationOr("JWT_REFRESH_TOKEN_DURATION", 7*24*time.Hour),
			Issuer:               envOr("JWT_ISSUER", "gorest"),
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
