package server

import (
	"crypto/subtle"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

const (
	apiKeySchemeName = "apiKey"
	apiKeyHeader     = "X-API-Key"
)

// apiKeyAuth guards every huma operation. huma middleware only runs for
// registered operations, so /docs, /swagger and /openapi.json stay open.
// key must be non-empty (Run enforces it): an empty key would equal a
// missing header and let everything through.
func apiKeyAuth(api huma.API, key string) func(huma.Context, func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		if subtle.ConstantTimeCompare([]byte(ctx.Header(apiKeyHeader)), []byte(key)) != 1 {
			_ = huma.WriteErr(api, ctx, http.StatusUnauthorized, "invalid or missing "+apiKeyHeader)
			return
		}
		next(ctx)
	}
}
