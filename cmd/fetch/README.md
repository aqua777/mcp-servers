# Fetch Command

## Overview
The `fetch` command is a Go entrypoint that wires the runtime tooling in `github.com/aqua777/mcp-servers/core/pkg` to a CLI-friendly executable. It:

1. Parses a small set of network-tuning flags (`--user-agent`, `--ignore-robots-txt`, `--proxy-url`).
2. Sets up a cancellable context and listens for `SIGINT`/`SIGTERM` so long-running fetches exit gracefully.
3. Builds a `fetch.Options` struct from the parsed flags.
4. Hands control to `runtime.Run(ctx, "fetch", opts)` which loads the `fetch` tool implementation from the shared runtime and executes it with the configured options.
5. Reports any runtime errors to `stderr` and returns a non-zero exit code so calling scripts can detect failures.

## Flag Details
- `--user-agent string`: overrides the default User-Agent header sent with HTTP requests.
- `--ignore-robots-txt`: when true, instructs the underlying fetch tool to skip robots.txt compliance checks (use with caution).
- `--proxy-url string`: routes outgoing requests through the provided proxy URL.

## Signal Handling & Cancellation
Immediately after parsing flags, `main` creates a `context.WithCancel` and registers for `SIGINT` and `SIGTERM`. When either signal arrives, the goroutine cancels the context, giving the runtime a chance to unwind in-flight work before the process exits.

## Runtime Integration
`runtime.Run` is the shared entrypoint used across MCP servers. For this command it is invoked with the tool key `"fetch"` and the `fetch.Options` built from the CLI flags. The runtime is responsible for:

- Loading the fetch tool implementation.
- Passing along the structured options.
- Managing lifecycle hooks, logging, and other cross-tool concerns.

Any error returned from `runtime.Run` is printed and causes the process to exit with status code `1` so upstream automation can treat the command as failed.
