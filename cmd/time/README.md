# Time Server

An MCP server that provides tools for retrieving current times and converting between timezones.

## Overview

This is a Go-native implementation of the reference Time MCP server. It provides tools to interact with time across global timezones.

### Tools

- `get_current_time`: Get current time in a specific timezone (e.g., 'America/New_York', 'Europe/London').
- `convert_time`: Convert a specific time (HH:MM) from one timezone to another.

## Building

From the `golang` directory:

```bash
go build -o time-server ./cmd/time
```

## Running

```bash
./time-server
```

## Example Usage (with mcp inspector)

```bash
npx @modelcontextprotocol/inspector ./time-server
```
