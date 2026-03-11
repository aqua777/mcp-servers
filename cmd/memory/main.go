package main

import (
	"context"
	"fmt"
	"os"

	"github.com/aqua777/krait"
	"github.com/aqua777/mcp-servers/common"
	"github.com/aqua777/mcp-servers/core/pkg/runtime"
	"github.com/aqua777/mcp-servers/core/pkg/tools/memory"
)

func runMemoryServer(args []string) error {
	memoryFilePath := krait.GetString("app.memory-file-path")

	// The memory manager handles empty string (defaults to memory.jsonl in executable directory)
	opts := memory.Options{
		MemoryFilePath: memoryFilePath,
	}

	ctx := context.Background()

	if err := runtime.Run(ctx, common.MCP_Memory, opts); err != nil {
		return fmt.Errorf("error running memory server: %w", err)
	}

	return nil
}

func main() {
	app := krait.App("memory-server", "Memory MCP Server", "An MCP server that provides a simple file-based knowledge graph for storing entities, relations, and observations.").
		WithConfig("", "config", "c", "APP_CONFIG").
		WithStringP("app.memory-file-path", "Path to the memory.jsonl file", "memory-file-path", "m", "MEMORY_FILE_PATH", "").
		WithRun(runMemoryServer)

	if err := app.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %v\n", err)
		os.Exit(1)
	}
}
