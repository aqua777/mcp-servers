# Everything MCP Server (Go)

This is a Go port of the reference "Everything" MCP server. It attempts to exercise various features of the MCP protocol including tools, resources, prompts, and server-side notifications (simulated logging/resource updates).

## Usage

You can run the server using different transports by passing an argument to the binary.

### STDIO (Default)
```bash
go build -o everything .
./everything
# or explicitly:
./everything stdio
```

### SSE Transport
Starts an HTTP server on port `3001` serving SSE on `/sse` and receiving messages on `/message`.
```bash
./everything sse
```

### Streamable HTTP Transport
Starts an HTTP server on port `3001` serving the Streamable HTTP transport on `/mcp`.
```bash
./everything streamableHttp
```

## Features Demonstrated
* **Tools**: `echo`, `get-env`, `get-sum`, `get-tiny-image`, `get-annotated-message`, `get-structured-content`
* **Prompts**: `simple-prompt`, `args-prompt`
* **Resources**: Demonstrates `demo://resource/dynamic/text/{id}` and `demo://resource/dynamic/blob/{id}` URI templates logic
* **Server Notifications**: Background goroutines that simulate logging messages and resource updates.
