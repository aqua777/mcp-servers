package main

import (
	"context"
	"fmt"
	"os"
	_ "time/tzdata"

	"github.com/aqua777/krait"
	"github.com/aqua777/mcp-servers/common"
	"github.com/aqua777/mcp-servers/core/pkg/runtime"
	_ "github.com/aqua777/mcp-servers/core/pkg/tools/time"
)

func runTimeServer(args []string) error {
	ctx := context.Background()

	// Using nil for opts as time server doesn't take special options
	if err := runtime.Run(ctx, common.MCP_Time, nil); err != nil {
		return fmt.Errorf("error running time server: %w", err)
	}

	return nil
}

func main() {
	app := krait.App(common.MCP_Time, "Time MCP Server", "An MCP server that provides time retrieval and timezone conversion capabilities.").
		WithRun(runTimeServer)

	if err := app.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %v\n", err)
		os.Exit(1)
	}
}
