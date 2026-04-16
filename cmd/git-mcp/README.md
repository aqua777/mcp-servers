# Git MCP Server (Golang)

A Go implementation of the Git MCP Server, mirroring the reference Python implementation. Provides tools to read, search, and manipulate Git repositories via Large Language Models.

## Features

- **Full Git workflow** via MCP tools: status, diff, add, commit, reset, log, branches, checkout, show
- **Dual output formats** - All read operations support both text (Git CLI style) and JSON (structured metadata)
- **AI-first mode** - Optimized defaults for AI consumption: JSON output + structured errors
- **Advanced diff controls** - Toggle line-level changes, limit file count, get context before/after
- **Enhanced metadata** - Ahead/behind computation, detached HEAD detection, untracked file types
- **Security** - Repository path restriction with symlink-safe validation, flag-injection protection
- **Pure-Go** implementation using [go-git](https://github.com/go-git/go-git) — no system `git` binary required

## Configuration

| Flag | Short | Env Var | Description |
|------|-------|---------|-------------|
| `--repository` | `-r` | `GIT_REPOSITORY` | Restrict all operations to a specific repository path. When set, any `repo_path` argument outside this directory is rejected. |
| `--output` | `-o` | `GIT_OUTPUT_FORMAT` | Default output format: `text` or `json` (default: `text`) |
| `--ai-mode` | `-a` | `GIT_AI_MODE` | Enable AI-first mode: defaults to JSON output and structured error responses |

## Tools Exposed

| Tool | Description |
|------|-------------|
| `git_status` | Shows the working tree status |
| `git_diff_unstaged` | Shows unstaged changes in the working directory |
| `git_diff_staged` | Shows changes staged for commit |
| `git_diff` | Shows differences between branches or commits |
| `git_commit` | Records staged changes to the repository |
| `git_add` | Adds file contents to the staging area |
| `git_reset` | Unstages all staged changes (mixed reset) |
| `git_log` | Shows commit logs with optional date filtering |
| `git_create_branch` | Creates a new branch from an optional base branch |
| `git_checkout` | Switches branches |
| `git_show` | Shows the contents of a commit (metadata + diff) |
| `git_branch` | Lists branches (local/remote/all) with optional contains/not-contains filters |

### Tool Parameters

All read-only tools support a `format` parameter to override the server default:

```json
{
  "repo_path": "/path/to/repo",
  "format": "json"  // or "text"
}
```

**Diff operations** (`git_diff_unstaged`, `git_diff_staged`, `git_diff`, `git_show`) support additional parameters:

- **`include_diff_content`** (boolean, default: `true`) - When `false`, returns file-level metadata only (path, status, additions, deletions) without line-by-line changes. Useful for getting an overview of large diffs.
- **`max_files`** (integer, default: `0` = unlimited) - Limits the number of files included in the diff. When truncated, the response includes `"truncated": true`.
- **`context_lines`** (integer, default: `3`) - Number of context lines around changes in unified diff format.

Example:
```json
{
  "repo_path": "/path/to/repo",
  "format": "json",
  "include_diff_content": false,
  "max_files": 10
}
```

## Usage

```bash
# Build
go build -o /tmp/git-mcp ./cmd/git-mcp

# Run (unrestricted — any repo path is accepted)
/tmp/git-mcp

# Run restricted to a specific repository
/tmp/git-mcp --repository /path/to/my/repo

# Run in AI mode (JSON output + structured errors)
/tmp/git-mcp --ai-mode

# Run with explicit JSON output
/tmp/git-mcp --output json

# Or via environment variables
GIT_REPOSITORY=/path/to/my/repo GIT_AI_MODE=true /tmp/git-mcp
```

## IDE Configuration

Add to your `mcp_config.json` (Windsurf) or `claude_desktop_config.json` (Claude Desktop / Claude Code):

```json
{
  "mcpServers": {
    "git": {
      "command": "/tmp/git-mcp",
      "args": []
    }
  }
}
```

To restrict operations to a specific repository:

```json
{
  "mcpServers": {
    "git": {
      "command": "/tmp/git-mcp",
      "args": ["--repository", "/path/to/my/repo"]
    }
  }
}
```

Or use the environment variable instead of a flag:

```json
{
  "mcpServers": {
    "git": {
      "command": "/tmp/git-mcp",
      "args": [],
      "env": {
        "GIT_REPOSITORY": "/path/to/my/repo"
      }
    }
  }
}
```

For AI-optimized output (recommended for LLM clients):

```json
{
  "mcpServers": {
    "git": {
      "command": "/tmp/git-mcp",
      "args": ["--ai-mode", "--repository", "/path/to/my/repo"]
    }
  }
}
```

## MCP Inspector

```bash
npx @modelcontextprotocol/inspector /tmp/git-mcp
```

## Examples

See [`../../examples/git-mcp`](../../examples/git-mcp) for a demo of interacting with this server using the Go MCP SDK.
