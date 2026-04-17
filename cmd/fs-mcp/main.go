package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/aqua777/krait"
	"github.com/aqua777/mcp-servers/common"
	"github.com/aqua777/mcp-servers/core/pkg/runtime"
	"github.com/aqua777/mcp-servers/core/pkg/tools/filesystem"
)

func runFilesystemServer(args []string) error {
	allowedDirectories := krait.GetStringSlice("app.allowed-directories")
	outputFormat := krait.GetString("app.output")
	aiMode := krait.GetBool("app.ai-mode")

	if len(allowedDirectories) == 0 {
		return fmt.Errorf("at least one allowed directory must be specified")
	}

	// Convert to absolute paths
	for i, dir := range allowedDirectories {
		absDir, err := filepath.Abs(dir)
		if err != nil {
			return fmt.Errorf("error resolving absolute path for %s: %w", dir, err)
		}
		allowedDirectories[i] = absDir
	}

	opts := filesystem.Options{
		AllowedDirectories: allowedDirectories,
		OutputFormat:       outputFormat,
		AIMode:             aiMode,
	}

	ctx := context.Background()
	if err := runtime.Run(ctx, common.MCP_FileSystem, opts); err != nil {
		return fmt.Errorf("error running filesystem server: %w", err)
	}
	return nil
}

func main() {
	app := krait.App(common.MCP_FileSystem, "Filesystem MCP Server", "An MCP server that provides sandboxed filesystem access tools.").
		WithStringSliceP("app.allowed-directories", "Allowed directories for filesystem access", "allowed-directories", "", "", []string{}).
		WithStringP("app.output", "Default output format: text or json", "output", "o", "FS_OUTPUT_FORMAT", "text").
		WithBoolP("app.ai-mode", "Enable AI-first mode (JSON output, structured errors)", "ai-mode", "a", "FS_AI_MODE", false).
		WithRun(runFilesystemServer)

	if err := app.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %v\n", err)
		os.Exit(1)
	}
}
