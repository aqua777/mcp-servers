package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/aqua777/krait"
	"github.com/aqua777/mcp-servers/common"
	"github.com/aqua777/mcp-servers/core/pkg/runtime"
	"github.com/aqua777/mcp-servers/core/pkg/tools/everything"
)

const (
	transportFlagName = "transport"
)

type serverConfig struct {
	Transport              string `mapstructure:"transport"`
	Port                   int    `mapstructure:"port"`
	GzipMaxFetchSize       int    `mapstructure:"gzip-max-fetch-size"`
	GzipMaxFetchTimeMillis int    `mapstructure:"gzip-max-fetch-time-millis"`
	GzipAllowedDomains     string `mapstructure:"gzip-allowed-domains"`
}

func runEverythingServer(args []string) error {
	ctx := context.Background()
	cfg := serverConfig{
		Transport:              "stdio",
		Port:                   3001,
		GzipMaxFetchSize:       10 * 1024 * 1024,
		GzipMaxFetchTimeMillis: 30000,
		GzipAllowedDomains:     "",
	}

	if len(args) > 0 {
		cfg.Transport = args[0]
	}

	opts := everything.Options{
		GzipMaxFetchSize:       cfg.GzipMaxFetchSize,
		GzipMaxFetchTimeMillis: cfg.GzipMaxFetchTimeMillis,
		GzipAllowedDomains:     cfg.GzipAllowedDomains,
	}

	switch cfg.Transport {
	case "stdio":
		if err := runtime.Run(ctx, common.MCP_Everything, opts); err != nil {
			return fmt.Errorf("error running everything server (stdio): %w", err)
		}
	case "sse":
		if err := runSSEServer(cfg.Port, opts); err != nil {
			return fmt.Errorf("error running everything server (sse): %w", err)
		}
	case "streamableHttp":
		if err := runStreamableHTTPServer(cfg.Port, opts); err != nil {
			return fmt.Errorf("error running everything server (streamableHttp): %w", err)
		}
	default:
		return fmt.Errorf("unknown transport %s; available transports: stdio, sse, streamableHttp", cfg.Transport)
	}

	return nil
}

func main() {
	app := krait.App("everything-server", "Everything MCP Server", "A Go port of the Everything reference server.").
		WithConfig("", "config", "c", "APP_CONFIG").
		WithIntP("app.port", "Port to listen on for SSE/HTTP transports", "port", "p", "PORT", 3001).
		WithInt("app.gzip-max-fetch-size", "Maximum fetch size in bytes for gzip tool", "gzip-max-fetch-size", "GZIP_MAX_FETCH_SIZE", 10*1024*1024).
		WithInt("app.gzip-max-fetch-time-millis", "Maximum fetch time in milliseconds for gzip tool", "gzip-max-fetch-time-millis", "GZIP_MAX_FETCH_TIME_MILLIS", 30000).
		WithString("app.gzip-allowed-domains", "Comma-separated allowlist of domains for gzip tool", "gzip-allowed-domains", "GZIP_ALLOWED_DOMAINS", "").
		WithRun(runEverythingServer)

	if err := app.Execute(); err != nil {
		if !errors.Is(err, context.Canceled) {
			fmt.Fprintf(os.Stderr, "Fatal error: %v\n", err)
		}
		os.Exit(1)
	}
}
