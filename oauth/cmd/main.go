// Standalone OAuth2 authorization-code demo server, split out of huma-bob
// (which now only demonstrates huma+bob CRUD) so the two aren't tangled.
// Uses huma (https://github.com/danielgtaylor/huma) with the standard
// library net/http.ServeMux as the router.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/manhrev/gorest/oauth/internal/server"
	applog "github.com/manhrev/gorest/common/log"
)

func main() {
	// SIGINT/SIGTERM cancels ctx, which server.Run uses to trigger graceful
	// shutdown (stop new requests, drain in-flight, flush tracing).
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := server.Run(ctx); err != nil {
		applog.Bootstrap().Error("fatal", "error", err)
		os.Exit(1)
	}
}
