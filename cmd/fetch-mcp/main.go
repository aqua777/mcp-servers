package main

import (
	"context"
	"fmt"
	"os"

	"github.com/aqua777/krait"
	"github.com/aqua777/mcp-servers/common"
	"github.com/aqua777/mcp-servers/core/pkg/runtime"
	"github.com/aqua777/mcp-servers/core/pkg/tools/fetch"
)

func runFetchServer(args []string) error {
	userAgent := krait.GetString("app.user-agent")
	ignoreRobotsTxt := krait.GetBool("app.ignore-robots-txt")
	proxyURL := krait.GetString("app.proxy-url")

	opts := fetch.Options{
		CustomUserAgent: userAgent,
		IgnoreRobotsTxt: ignoreRobotsTxt,
		ProxyURL:        proxyURL,
	}

	ctx := context.Background()
	if err := runtime.Run(ctx, common.MCP_Fetch, opts); err != nil {
		return fmt.Errorf("error running fetch server: %w", err)
	}
	return nil
}

func main() {
	app := krait.App(common.MCP_Fetch, "Fetch MCP Server", "An MCP server that fetches URLs and returns content as markdown with robots.txt enforcement.").
		WithConfig("", "config", "c", "APP_CONFIG").
		WithStringP("app.user-agent", "Custom User-Agent string", "user-agent", "", "", "").
		WithBoolP("app.ignore-robots-txt", "Ignore robots.txt restrictions", "ignore-robots-txt", "", "", false).
		WithStringP("app.proxy-url", "Proxy URL to use for requests", "proxy-url", "", "", "").
		WithRun(runFetchServer)

	if err := app.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %v\n", err)
		os.Exit(1)
	}
}
