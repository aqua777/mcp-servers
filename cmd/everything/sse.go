package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/aqua777/mcp-servers/core/pkg/tools/everything"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func runSSEServer(port int, opts everything.Options) error {
	ctx := context.Background()

	sseHandler := mcp.NewSSEHandler(func(_ *http.Request) *mcp.Server {
		server, err := everything.NewServer(ctx, opts)
		if err != nil {
			log.Printf("Failed to initialize everything server for session: %v", err)
			return nil
		}
		return server
	}, nil)

	// Add basic CORS middleware since TS version used Express CORS
	corsMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			next.ServeHTTP(w, r)
		})
	}

	loggingMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Printf("Received %s request to %s", r.Method, r.URL.Path)
			next.ServeHTTP(w, r)
		})
	}

	mux := http.NewServeMux()
	mux.Handle("/sse", loggingMiddleware(corsMiddleware(sseHandler)))
	mux.Handle("/message", loggingMiddleware(corsMiddleware(sseHandler)))

	addr := fmt.Sprintf(":%d", port)
	log.Printf("Starting SSE server on %s", addr)

	if err := http.ListenAndServe(addr, mux); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("http server failed: %w", err)
	}

	return nil
}
