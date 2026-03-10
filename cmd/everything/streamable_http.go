package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/aqua777/mcp-servers/core/pkg/tools/everything"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func runStreamableHTTPServer(port int, opts everything.Options) error {
	ctx := context.Background()

	// In the real reference implementation they maintain
	// an in-memory event store for resumability and use a StreamableHTTPHandler.
	httpHandler := mcp.NewStreamableHTTPHandler(func(_ *http.Request) *mcp.Server {
		server, err := everything.NewServer(ctx, opts)
		if err != nil {
			log.Printf("Failed to initialize everything server for session: %v", err)
			return nil
		}
		return server
	}, &mcp.StreamableHTTPOptions{
		EventStore: mcp.NewMemoryEventStore(&mcp.MemoryEventStoreOptions{}),
	})

	// Add basic CORS middleware since TS version used Express CORS
	corsMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, mcp-session-id, last-event-id")
			w.Header().Set("Access-Control-Expose-Headers", "mcp-session-id, last-event-id, mcp-protocol-version")
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}

	loggingMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Printf("Received MCP %s request", r.Method)
			next.ServeHTTP(w, r)
		})
	}

	mux := http.NewServeMux()
	mux.Handle("/mcp", loggingMiddleware(corsMiddleware(httpHandler)))

	addr := fmt.Sprintf(":%d", port)
	log.Printf("Starting Streamable HTTP server on %s", addr)

	if err := http.ListenAndServe(addr, mux); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("http server failed: %w", err)
	}

	return nil
}
