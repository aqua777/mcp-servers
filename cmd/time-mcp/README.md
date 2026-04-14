# Time Server

An MCP server that provides tools for retrieving current times and converting between timezones.

## Overview

This is a Go-native implementation of the reference Time MCP server. It provides tools to interact with time across global timezones.

### Tools

- `get_current_time`: Get current time in a specific timezone (e.g., 'America/New_York', 'Europe/London').
- `convert_time`: Convert a specific time (HH:MM) from one timezone to another.

## Usage

```bash
# Build
go build -o /tmp/time-mcp ./cmd/time-mcp

# Run
/tmp/time-mcp
```

## IDE Configuration

Add to your `mcp_config.json` (Windsurf) or `claude_desktop_config.json` (Claude Desktop / Claude Code):

```json
{
  "mcpServers": {
    "time": {
      "command": "/tmp/time-mcp",
      "args": []
    }
  }
}
```

## MCP Inspector

```bash
npx @modelcontextprotocol/inspector /tmp/time-mcp
```

## Examples

See [`../../examples/time-mcp`](../../examples/time-mcp) for a demo of interacting with this server using the Go MCP SDK.
