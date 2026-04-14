# Filesystem MCP Server (Golang)

A Go implementation of the Filesystem MCP Server. Provides sandboxed read/write/edit/search tools restricted to one or more allowed directories.

## Features

- Read files (text, media, multiple at once)
- Write and move files
- List directories and directory trees
- Search files by pattern
- Edit files with dry-run diff preview
- File metadata (`get_file_info`, `list_allowed_directories`)
- Secure path validation — all paths are confined to the declared allowed directories

## Usage

One or more allowed directories must be passed as positional arguments:

```bash
# Build
go build -o /tmp/fs-mcp ./cmd/fs-mcp

# Grant access to a single directory
/tmp/fs-mcp /home/user/projects

# Grant access to multiple directories
/tmp/fs-mcp /home/user/projects /home/user/notes
```

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
| `search_files` | Search for files matching a glob pattern |
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

See [`../../examples/fs-mcp`](../../examples/fs-mcp) for a demo of interacting with this server using the Go MCP SDK.
