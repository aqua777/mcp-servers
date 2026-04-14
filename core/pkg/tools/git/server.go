package git

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	gogit "github.com/go-git/go-git/v5"
	"github.com/aqua777/mcp-servers/common"
	"github.com/aqua777/mcp-servers/core/pkg/runtime"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func init() {
	runtime.Register(common.MCP_Git, NewServer)
}

// GitServer holds the MCP server and its configuration.
type GitServer struct {
	server  *mcp.Server
	options Options
}

// NewServer creates and registers the Git MCP server.
func NewServer(ctx context.Context, opts any) (*mcp.Server, error) {
	var options Options
	if opts != nil {
		var ok bool
		options, ok = opts.(Options)
		if !ok {
			return nil, fmt.Errorf("git server: invalid options type %T", opts)
		}
	}

	gs := &GitServer{options: options}
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "mcp-git",
		Version: "1.0.0",
	}, &mcp.ServerOptions{})
	gs.server = server

	server.AddTool(&mcp.Tool{
		Name:        ToolGitStatus,
		Description: "Shows the working tree status",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"repo_path": map[string]any{"type": "string", "description": "Path to Git repository"},
			},
			"required": []string{"repo_path"},
		},
	}, gs.handleGitStatus)

	server.AddTool(&mcp.Tool{
		Name:        ToolGitDiffUnstaged,
		Description: "Shows changes in the working directory that are not yet staged",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"repo_path":     map[string]any{"type": "string", "description": "Path to Git repository"},
				"context_lines": map[string]any{"type": "integer", "description": "Number of context lines to show (default: 3)"},
			},
			"required": []string{"repo_path"},
		},
	}, gs.handleGitDiffUnstaged)

	server.AddTool(&mcp.Tool{
		Name:        ToolGitDiffStaged,
		Description: "Shows changes that are staged for commit",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"repo_path":     map[string]any{"type": "string", "description": "Path to Git repository"},
				"context_lines": map[string]any{"type": "integer", "description": "Number of context lines to show (default: 3)"},
			},
			"required": []string{"repo_path"},
		},
	}, gs.handleGitDiffStaged)

	server.AddTool(&mcp.Tool{
		Name:        ToolGitDiff,
		Description: "Shows differences between branches or commits",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"repo_path":     map[string]any{"type": "string", "description": "Path to Git repository"},
				"target":        map[string]any{"type": "string", "description": "Target branch or commit to compare with"},
				"context_lines": map[string]any{"type": "integer", "description": "Number of context lines to show (default: 3)"},
			},
			"required": []string{"repo_path", "target"},
		},
	}, gs.handleGitDiff)

	server.AddTool(&mcp.Tool{
		Name:        ToolGitCommit,
		Description: "Records changes to the repository",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"repo_path": map[string]any{"type": "string", "description": "Path to Git repository"},
				"message":   map[string]any{"type": "string", "description": "Commit message"},
			},
			"required": []string{"repo_path", "message"},
		},
	}, gs.handleGitCommit)

	server.AddTool(&mcp.Tool{
		Name:        ToolGitAdd,
		Description: "Adds file contents to the staging area",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"repo_path": map[string]any{"type": "string", "description": "Path to Git repository"},
				"files": map[string]any{
					"type":        "array",
					"items":       map[string]any{"type": "string"},
					"description": "Array of file paths to stage",
				},
			},
			"required": []string{"repo_path", "files"},
		},
	}, gs.handleGitAdd)

	server.AddTool(&mcp.Tool{
		Name:        ToolGitReset,
		Description: "Unstages all staged changes",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"repo_path": map[string]any{"type": "string", "description": "Path to Git repository"},
			},
			"required": []string{"repo_path"},
		},
	}, gs.handleGitReset)

	server.AddTool(&mcp.Tool{
		Name:        ToolGitLog,
		Description: "Shows the commit logs",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"repo_path":       map[string]any{"type": "string", "description": "Path to Git repository"},
				"max_count":       map[string]any{"type": "integer", "description": "Maximum number of commits to show (default: 10)"},
				"start_timestamp": map[string]any{"type": "string", "description": "Start timestamp for filtering commits (ISO 8601, relative dates like '2 weeks ago', or absolute dates)"},
				"end_timestamp":   map[string]any{"type": "string", "description": "End timestamp for filtering commits (ISO 8601, relative dates, or absolute dates)"},
			},
			"required": []string{"repo_path"},
		},
	}, gs.handleGitLog)

	server.AddTool(&mcp.Tool{
		Name:        ToolGitCreateBranch,
		Description: "Creates a new branch from an optional base branch",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"repo_path":   map[string]any{"type": "string", "description": "Path to Git repository"},
				"branch_name": map[string]any{"type": "string", "description": "Name of the new branch"},
				"base_branch": map[string]any{"type": "string", "description": "Base branch to create from (defaults to current branch)"},
			},
			"required": []string{"repo_path", "branch_name"},
		},
	}, gs.handleGitCreateBranch)

	server.AddTool(&mcp.Tool{
		Name:        ToolGitCheckout,
		Description: "Switches branches",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"repo_path":   map[string]any{"type": "string", "description": "Path to Git repository"},
				"branch_name": map[string]any{"type": "string", "description": "Name of branch to checkout"},
			},
			"required": []string{"repo_path", "branch_name"},
		},
	}, gs.handleGitCheckout)

	server.AddTool(&mcp.Tool{
		Name:        ToolGitShow,
		Description: "Shows the contents of a commit",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"repo_path": map[string]any{"type": "string", "description": "Path to Git repository"},
				"revision":  map[string]any{"type": "string", "description": "The revision (commit hash, branch name, tag) to show"},
			},
			"required": []string{"repo_path", "revision"},
		},
	}, gs.handleGitShow)

	server.AddTool(&mcp.Tool{
		Name:        ToolGitBranch,
		Description: "List Git branches",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"repo_path":    map[string]any{"type": "string", "description": "Path to the Git repository"},
				"branch_type":  map[string]any{"type": "string", "enum": []string{"local", "remote", "all"}, "description": "Whether to list local branches ('local'), remote branches ('remote') or all branches ('all')"},
				"contains":     map[string]any{"type": "string", "description": "Commit SHA that branch should contain"},
				"not_contains": map[string]any{"type": "string", "description": "Commit SHA that branch should NOT contain"},
			},
			"required": []string{"repo_path", "branch_type"},
		},
	}, gs.handleGitBranch)

	return server, nil
}

// openRepo validates the path and opens the git repository.
func (gs *GitServer) openRepo(repoPath string) (*gogit.Repository, error) {
	if err := validateRepoPath(repoPath, gs.options.AllowedRepository); err != nil {
		return nil, err
	}
	repo, err := gogit.PlainOpen(repoPath)
	if err != nil {
		return nil, fmt.Errorf("opening repository at %q: %w", repoPath, err)
	}
	return repo, nil
}

func (gs *GitServer) handleGitStatus(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args struct {
		RepoPath string `json:"repo_path"`
	}
	if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
		return handleError(err)
	}
	repo, err := gs.openRepo(args.RepoPath)
	if err != nil {
		return handleError(err)
	}
	result, err := gitStatus(repo)
	if err != nil {
		return handleError(err)
	}
	return handleSuccess("Repository status:\n" + result)
}

func (gs *GitServer) handleGitDiffUnstaged(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args struct {
		RepoPath     string `json:"repo_path"`
		ContextLines *int   `json:"context_lines"`
	}
	if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
		return handleError(err)
	}
	contextLines := DefaultContextLines
	if args.ContextLines != nil {
		contextLines = *args.ContextLines
	}
	repo, err := gs.openRepo(args.RepoPath)
	if err != nil {
		return handleError(err)
	}
	result, err := gitDiffUnstaged(repo, contextLines)
	if err != nil {
		return handleError(err)
	}
	return handleSuccess("Unstaged changes:\n" + result)
}

func (gs *GitServer) handleGitDiffStaged(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args struct {
		RepoPath     string `json:"repo_path"`
		ContextLines *int   `json:"context_lines"`
	}
	if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
		return handleError(err)
	}
	contextLines := DefaultContextLines
	if args.ContextLines != nil {
		contextLines = *args.ContextLines
	}
	repo, err := gs.openRepo(args.RepoPath)
	if err != nil {
		return handleError(err)
	}
	result, err := gitDiffStaged(repo, contextLines)
	if err != nil {
		return handleError(err)
	}
	return handleSuccess("Staged changes:\n" + result)
}

func (gs *GitServer) handleGitDiff(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args struct {
		RepoPath     string `json:"repo_path"`
		Target       string `json:"target"`
		ContextLines *int   `json:"context_lines"`
	}
	if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
		return handleError(err)
	}
	if args.Target == "" {
		return handleError(fmt.Errorf("missing required argument: target"))
	}
	contextLines := DefaultContextLines
	if args.ContextLines != nil {
		contextLines = *args.ContextLines
	}
	repo, err := gs.openRepo(args.RepoPath)
	if err != nil {
		return handleError(err)
	}
	result, err := gitDiff(repo, args.Target, contextLines)
	if err != nil {
		return handleError(err)
	}
	return handleSuccess(fmt.Sprintf("Diff with %s:\n%s", args.Target, result))
}

func (gs *GitServer) handleGitCommit(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args struct {
		RepoPath string `json:"repo_path"`
		Message  string `json:"message"`
	}
	if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
		return handleError(err)
	}
	if args.Message == "" {
		return handleError(fmt.Errorf("missing required argument: message"))
	}
	repo, err := gs.openRepo(args.RepoPath)
	if err != nil {
		return handleError(err)
	}
	result, err := gitCommit(repo, args.Message)
	if err != nil {
		return handleError(err)
	}
	return handleSuccess(result)
}

func (gs *GitServer) handleGitAdd(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args struct {
		RepoPath string   `json:"repo_path"`
		Files    []string `json:"files"`
	}
	if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
		return handleError(err)
	}
	if len(args.Files) == 0 {
		return handleError(fmt.Errorf("missing required argument: files"))
	}
	repo, err := gs.openRepo(args.RepoPath)
	if err != nil {
		return handleError(err)
	}
	result, err := gitAdd(repo, args.Files)
	if err != nil {
		return handleError(err)
	}
	return handleSuccess(result)
}

func (gs *GitServer) handleGitReset(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args struct {
		RepoPath string `json:"repo_path"`
	}
	if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
		return handleError(err)
	}
	repo, err := gs.openRepo(args.RepoPath)
	if err != nil {
		return handleError(err)
	}
	result, err := gitReset(repo)
	if err != nil {
		return handleError(err)
	}
	return handleSuccess(result)
}

func (gs *GitServer) handleGitLog(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args struct {
		RepoPath       string `json:"repo_path"`
		MaxCount       *int   `json:"max_count"`
		StartTimestamp string `json:"start_timestamp"`
		EndTimestamp   string `json:"end_timestamp"`
	}
	if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
		return handleError(err)
	}
	maxCount := 10
	if args.MaxCount != nil {
		maxCount = *args.MaxCount
	}
	repo, err := gs.openRepo(args.RepoPath)
	if err != nil {
		return handleError(err)
	}
	entries, err := gitLog(repo, maxCount, args.StartTimestamp, args.EndTimestamp)
	if err != nil {
		return handleError(err)
	}
	return handleSuccess("Commit history:\n" + strings.Join(entries, "\n"))
}

func (gs *GitServer) handleGitCreateBranch(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args struct {
		RepoPath   string `json:"repo_path"`
		BranchName string `json:"branch_name"`
		BaseBranch string `json:"base_branch"`
	}
	if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
		return handleError(err)
	}
	if args.BranchName == "" {
		return handleError(fmt.Errorf("missing required argument: branch_name"))
	}
	repo, err := gs.openRepo(args.RepoPath)
	if err != nil {
		return handleError(err)
	}
	result, err := gitCreateBranch(repo, args.BranchName, args.BaseBranch)
	if err != nil {
		return handleError(err)
	}
	return handleSuccess(result)
}

func (gs *GitServer) handleGitCheckout(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args struct {
		RepoPath   string `json:"repo_path"`
		BranchName string `json:"branch_name"`
	}
	if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
		return handleError(err)
	}
	if args.BranchName == "" {
		return handleError(fmt.Errorf("missing required argument: branch_name"))
	}
	repo, err := gs.openRepo(args.RepoPath)
	if err != nil {
		return handleError(err)
	}
	result, err := gitCheckout(repo, args.BranchName)
	if err != nil {
		return handleError(err)
	}
	return handleSuccess(result)
}

func (gs *GitServer) handleGitShow(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args struct {
		RepoPath string `json:"repo_path"`
		Revision string `json:"revision"`
	}
	if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
		return handleError(err)
	}
	if args.Revision == "" {
		return handleError(fmt.Errorf("missing required argument: revision"))
	}
	repo, err := gs.openRepo(args.RepoPath)
	if err != nil {
		return handleError(err)
	}
	result, err := gitShow(repo, args.Revision)
	if err != nil {
		return handleError(err)
	}
	return handleSuccess(result)
}

func (gs *GitServer) handleGitBranch(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args struct {
		RepoPath    string `json:"repo_path"`
		BranchType  string `json:"branch_type"`
		Contains    string `json:"contains"`
		NotContains string `json:"not_contains"`
	}
	if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
		return handleError(err)
	}
	if args.BranchType == "" {
		return handleError(fmt.Errorf("missing required argument: branch_type"))
	}
	repo, err := gs.openRepo(args.RepoPath)
	if err != nil {
		return handleError(err)
	}
	result, err := gitBranch(repo, args.BranchType, args.Contains, args.NotContains)
	if err != nil {
		return handleError(err)
	}
	return handleSuccess(result)
}

func handleSuccess(text string) (*mcp.CallToolResult, error) {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: text},
		},
	}, nil
}

func handleError(err error) (*mcp.CallToolResult, error) {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: fmt.Sprintf("Error: %v", err)},
		},
		IsError: true,
	}, nil
}
