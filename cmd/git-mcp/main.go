package main

import (
	"context"
	"fmt"
	"os"

	"github.com/aqua777/krait"
	"github.com/aqua777/mcp-servers/common"
	"github.com/aqua777/mcp-servers/core/pkg/runtime"
	gittools "github.com/aqua777/mcp-servers/core/pkg/tools/git"
)

func runGitServer(args []string) error {
	repository := krait.GetString("app.repository")
	outputFormat := krait.GetString("app.output")
	aiMode := krait.GetBool("app.ai-mode")

	opts := gittools.Options{
		AllowedRepository: repository,
		OutputFormat:      outputFormat,
		AIMode:            aiMode,
	}

	ctx := context.Background()
	if err := runtime.Run(ctx, common.MCP_Git, opts); err != nil {
		return fmt.Errorf("error running git server: %w", err)
	}
	return nil
}

func main() {
	app := krait.App(common.MCP_Git, "Git MCP Server", "An MCP server that provides tools to read, search, and manipulate Git repositories.").
		WithStringP("app.repository", "Restrict operations to a specific repository path", "repository", "r", "GIT_REPOSITORY", "").
		WithStringP("app.output", "Default output format: text or json", "output", "o", "GIT_OUTPUT_FORMAT", "text").
		WithBoolP("app.ai-mode", "Enable AI-first mode (JSON output, structured errors)", "ai-mode", "a", "GIT_AI_MODE", false).
		WithRun(runGitServer)

	if err := app.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %v\n", err)
		os.Exit(1)
	}
}
