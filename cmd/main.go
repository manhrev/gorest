// Simple example HTTP server using huma (https://github.com/danielgtaylor/huma)
// with the standard library net/http.ServeMux as the router.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/manhrev/gorest/internal/server"
	applog "github.com/manhrev/gorest/pkg/log"
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
