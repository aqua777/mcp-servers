# Filesystem MCP Server (Golang)

A Go implementation of the Filesystem MCP Server. Provides sandboxed read/write/edit/search tools restricted to one or more allowed directories.

## Features

- Read files (text, media, multiple at once)
- Write and move files
- List directories and directory trees
- Search files by glob pattern (`search_files`)
- Search file **contents** by regex or literal string (`grep`) — ripgrep-inspired
- Edit files with dry-run diff preview
- File metadata (`get_file_info`, `list_allowed_directories`)
- Secure path validation — all paths are confined to the declared allowed directories
- Dual output format: `text` (human-readable) or `json` (structured, AI-friendly)

## Installation

```bash
go install github.com/aqua777/mcp-servers/cmd/fs-mcp@latest
```

## Usage

One or more allowed directories must be passed via `--allowed-directories` (or `-d`):

```bash
# Build
go build -o /tmp/fs-mcp ./cmd/fs-mcp

# Grant access to a single directory
/tmp/fs-mcp --allowed-directories /home/user/projects

# Grant access to multiple directories
/tmp/fs-mcp --allowed-directories /home/user/projects --allowed-directories /home/user/notes

# Enable AI-first mode (JSON output by default, structured errors)
/tmp/fs-mcp --ai-mode --allowed-directories /home/user/projects

# Set default output format explicitly
/tmp/fs-mcp --output json --allowed-directories /home/user/projects
```

### Flags

| Flag | Short | Env | Default | Description |
|------|-------|-----|---------|-------------|
| `--allowed-directories` | `-d` | — | (required) | One or more directories the server may access |
| `--output` | `-o` | `FS_OUTPUT_FORMAT` | `text` | Default output format: `text` or `json` |
| `--ai-mode` | `-a` | `FS_AI_MODE` | `false` | Enable AI-first mode: JSON output + structured errors |

## Tools Exposed

| Tool | Description |
|------|-------------|
| `read_text_file` | Read a text file's contents |
| `read_media_file` | Read a media file as base64 |
| `read_multiple_files` | Read several files at once |
| `write_file` | Write content to a file |
| `create_directory` | Create a directory (including parents) |
| `move_file` | Move or rename a file |
| `list_directory` | List directory contents |
| `list_directory_with_sizes` | List directory contents with file sizes |
| `directory_tree` | Recursive directory tree |
| `search_files` | Search for files/directories matching a glob pattern |
| `grep` | Search file contents by regex or literal pattern (ripgrep-inspired) |
| `edit_file` | Apply string-replacement edits with optional dry-run diff |
| `get_file_info` | File metadata (size, modification time, permissions) |
| `list_allowed_directories` | List the configured allowed directories |

## IDE Configuration

Add to your `mcp_config.json` (Windsurf) or `claude_desktop_config.json` (Claude Desktop / Claude Code):

```json
{
  "mcpServers": {
    "filesystem": {
      "command": "/tmp/fs-mcp",
      "args": ["/home/user/projects"]
    }
  }
}
```

To allow access to multiple directories:

```json
{
  "mcpServers": {
    "filesystem": {
      "command": "/tmp/fs-mcp",
      "args": ["/home/user/projects", "/home/user/notes"]
    }
  }
}
```

## MCP Inspector

```bash
npx @modelcontextprotocol/inspector /tmp/fs-mcp /home/user/projects
```

## Examples

See [`../../examples/fs-mcp`](../../examples/fs-mcp) for a demo of interacting with this server using the Go MCP SDK and a local LLM via Ollama.

```bash
# Run the example against a directory
cd examples/fs-mcp
go run main.go --allowed-directories /home/user/projects "Find all TODO comments in Go files"
go run main.go --allowed-directories /home/user/projects "Search for error handling patterns using grep"
go run main.go --allowed-directories /home/user/projects "List the directory tree"
```

### `grep` Tool Examples

The `grep` tool supports regex content search across a directory or single file:

```jsonc
// Basic regex search (recursive)
{ "path": "/home/user/projects", "pattern": "TODO" }

// Literal string, case-insensitive
{ "path": "/home/user/projects", "pattern": "fixme", "fixedStrings": true, "ignoreCase": true }

// PCRE2 engine with lookahead
{ "path": "/home/user/projects", "pattern": "func(?=.*Handler)", "engine": "pcre2" }

// Restrict to Go files, show 2 lines of context
{ "path": "/home/user/projects", "pattern": "err != nil", "includePatterns": ["*.go"], "contextLines": 2 }

// Exclude vendored code, cap at 50 results, JSON output
{
  "path": "/home/user/projects",
  "pattern": "deprecated",
  "excludePatterns": ["vendor/**", "*.min.js"],
  "maxMatches": 50,
  "format": "json"
}
```

#### `grep` Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `path` | string | — | Directory or file to search |
| `pattern` | string | — | Search pattern (regex or literal) |
| `fixedStrings` | bool | `false` | Treat pattern as literal string |
| `ignoreCase` | bool | `false` | Case-insensitive match |
| `smartCase` | bool | `false` | Case-insensitive unless pattern has uppercase |
| `engine` | string | `re2` | Regex engine: `re2` (default) or `pcre2` |
| `includePatterns` | []string | — | Only search files matching these globs |
| `excludePatterns` | []string | — | Skip files matching these globs |
| `contextBefore` | int | `0` | Lines of context before each match |
| `contextAfter` | int | `0` | Lines of context after each match |
| `contextLines` | int | `0` | Symmetric context lines (before + after) |
| `maxMatches` | int | `1000` | Max matches to return; `0` = unlimited |
| `format` | string | server default | Output format: `text` or `json` |
