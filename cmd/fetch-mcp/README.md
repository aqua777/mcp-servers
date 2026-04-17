# Fetch MCP Server (Golang)

A Go implementation of the Fetch MCP Server. Fetches URLs and returns their content as markdown, with robots.txt enforcement and optional proxy support.

## Features

- Converts HTML pages to clean markdown via readability extraction
- Enforces `robots.txt` by default (bypass with `--ignore-robots-txt`)
- Optional HTTP/HTTPS proxy support
- Content chunking with `start_index` / `max_length` parameters
- Raw mode to skip HTML→markdown conversion

## Configuration

| Flag | Env Var | Description |
|------|---------|-------------|
| `--user-agent` | — | Override the default User-Agent header |
| `--ignore-robots-txt` | — | Skip robots.txt compliance checks |
| `--proxy-url` | — | Route all requests through this proxy URL |

## Tools Exposed

| Tool | Description |
|------|-------------|
| `fetch` | Fetch a URL and return its content as markdown (or raw) |

## Installation

```bash
go install github.com/aqua777/mcp-servers/cmd/fetch-mcp@latest
```

## Usage

```bash
# Build
go build -o /tmp/fetch-mcp ./cmd/fetch-mcp

# Run (default — robots.txt enforced)
/tmp/fetch-mcp

# Run ignoring robots.txt
/tmp/fetch-mcp --ignore-robots-txt

# Run with a proxy
/tmp/fetch-mcp --proxy-url "http://proxy.example.com:8080"
```

## IDE Configuration

Add to your `mcp_config.json` (Windsurf) or `claude_desktop_config.json` (Claude Desktop / Claude Code):

```json
{
  "mcpServers": {
    "fetch": {
      "command": "/tmp/fetch-mcp",
      "args": []
    }
  }
}
```

To disable robots.txt enforcement:

```json
{
  "mcpServers": {
    "fetch": {
      "command": "/tmp/fetch-mcp",
      "args": ["--ignore-robots-txt"]
    }
  }
}
```

To route through a proxy:

```json
{
  "mcpServers": {
    "fetch": {
      "command": "/tmp/fetch-mcp",
      "args": ["--proxy-url", "http://proxy.example.com:8080"]
    }
  }
}
```

## MCP Inspector

```bash
npx @modelcontextprotocol/inspector /tmp/fetch-mcp
```

## Examples

See [`../../examples/fetch-mcp`](../../examples/fetch-mcp) for a demo of interacting with this server using the Go MCP SDK.
