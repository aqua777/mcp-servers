package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/aqua777/mcp-servers/common"
	_ "github.com/aqua777/mcp-servers/core/pkg/tools/sequentialthinking"
	"github.com/aqua777/mcp-servers/core/pkg/runtime"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
	}()

	fmt.Println("Sequential Thinking MCP Server Example")
	fmt.Println("To use this server, connect to it using an MCP client.")
	fmt.Println("Try sending sequential thinking tool calls to iterate through a problem.")

	// Set standard MCP environment variables if needed or run the default configuration
	if err := runtime.Run(ctx, common.MCP_SequentialThinking, nil); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
