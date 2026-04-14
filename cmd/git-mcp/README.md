# Git MCP Server (Golang)

A Go implementation of the Git MCP Server, mirroring the reference Python implementation. Provides tools to read, search, and manipulate Git repositories via Large Language Models.

## Features

- Full Git workflow via MCP tools: status, diff, add, commit, reset, log, branches, checkout, show
- Optional repository path restriction with symlink-safe path validation
- Flag-injection protection (rejects ref names starting with `-`)
- Pure-Go implementation using [go-git](https://github.com/go-git/go-git) — no system `git` binary required

## Configuration

| Flag | Short | Env Var | Description |
|------|-------|---------|-------------|
| `--repository` | `-r` | `GIT_REPOSITORY` | Restrict all operations to a specific repository path. When set, any `repo_path` argument outside this directory is rejected. |

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

## Usage

```bash
# Build
go build -o /tmp/git-mcp ./cmd/git-mcp

# Run (unrestricted — any repo path is accepted)
/tmp/git-mcp

# Run restricted to a specific repository
/tmp/git-mcp --repository /path/to/my/repo

# Or via environment variable
GIT_REPOSITORY=/path/to/my/repo /tmp/git-mcp
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

## MCP Inspector

```bash
npx @modelcontextprotocol/inspector /tmp/git-mcp
```

## Examples

See [`../../examples/git-mcp`](../../examples/git-mcp) for a demo of interacting with this server using the Go MCP SDK.
