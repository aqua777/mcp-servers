package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/aqua777/mcp-servers/core/pkg/runtime"
	"github.com/aqua777/mcp-servers/core/pkg/tools/filesystem"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <allowed-directory> [allowed-directory...]\n", os.Args[0])
		os.Exit(1)
	}

	allowedDirectories := os.Args[1:]

	for i, dir := range allowedDirectories {
		absDir, err := filepath.Abs(dir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error resolving absolute path for %s: %v\n", dir, err)
			os.Exit(1)
		}
		allowedDirectories[i] = absDir
	}

	opts := filesystem.Options{
		AllowedDirectories: allowedDirectories,
	}

	ctx := context.Background()

	if err := runtime.Run(ctx, "filesystem", opts); err != nil {
		fmt.Fprintf(os.Stderr, "Error running filesystem server: %v\n", err)
		os.Exit(1)
	}
}
