package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/aqua777/mcp-servers/core/pkg/runtime"
	"github.com/aqua777/mcp-servers/core/pkg/tools/fetch"
)

func main() {
	var (
		userAgent       = flag.String("user-agent", "", "Custom User-Agent string")
		ignoreRobotsTxt = flag.Bool("ignore-robots-txt", false, "Ignore robots.txt restrictions")
		proxyURL        = flag.String("proxy-url", "", "Proxy URL to use for requests")
	)

	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
	}()

	opts := fetch.Options{
		CustomUserAgent: *userAgent,
		IgnoreRobotsTxt: *ignoreRobotsTxt,
		ProxyURL:        *proxyURL,
	}

	if err := runtime.Run(ctx, "fetch", opts); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
