# Go MCP Servers (Experimental)

> ⚠️ Experimental ports of the reference MCP servers. APIs and transports may change; verify behavior against `docs/parity.md` before relying on them.

## Goals
- Provide Go-native equivalents of the reference servers (Everything, Filesystem, Fetch, Git, Memory, Sequential Thinking, Time).
- Ship each server as a standalone binary (`cmd/<server>`) and an embeddable module with examples under `./examples/<server>`.
- Track feature parity and gaps explicitly via `docs/parity.md`.

## Workspace Layout
```
./go.work                # Multi-module workspace for core + examples
./core                   # Shared runtime, config, logging, registration helpers
./cmd/<server>           # Individual server binaries (go.mod per server)
./examples/<server>      # Embedding demos for each implemented server
./docs/parity.md         # Parity matrix and limitations
./Makefile               # Docker dev image + unit test wrapper
```

## Implementation Status

### ✅ Implemented
- **Everything** – stdio, SSE, and Streamable HTTP transports; demo tools/resources/prompts. See [`cmd/everything/README.md`](./cmd/everything/README.md).
- **Fetch** – HTTP client with robots.txt enforcement, readability → markdown pipeline, proxy support. See [`cmd/fetch-mcp/README.md`](./cmd/fetch-mcp/README.md) and [`examples/fetch-mcp`](./examples/fetch-mcp).
- **Filesystem** – Read/write/edit/search tools with sandboxing. See [`docs/parity.md#📁-filesystem-server`](./docs/parity.md#📁-filesystem-server) and [`examples/fs-mcp`](./examples/fs-mcp).
- **Memory** – JSONL-backed graph store with create/read/search/delete tools. See [`cmd/memory-mcp/README.md`](./cmd/memory-mcp/README.md) and [`examples/memory-mcp`](./examples/memory-mcp).
- **Sequential Thinking** – In-memory thought history tools. See [`cmd/sequentialthinking/README.md`](./cmd/sequentialthinking/README.md) and [`examples/sequentialthinking`](./examples/sequentialthinking).
- **Git** – Pure-Go git tools (status, diff, add, commit, reset, log, branch, checkout, show) with optional repository path restriction and flag-injection protection. See [`cmd/git-mcp/README.md`](./cmd/git-mcp/README.md) and [`examples/git-mcp`](./examples/git-mcp).
- **Time** – Current time and timezone conversion tools. See [`cmd/time-mcp/README.md`](./cmd/time-mcp/README.md) and [`examples/time-mcp`](./examples/time-mcp).

## Current Status (high-level)
- ✅ Workspace + shared runtime in `core`
- ✅ Servers above implemented with docs + examples; parity tracked in `docs/parity.md`

## Quick Start

### Requirements
- Go 1.24+
- Vendored modules included (`go work vendor`).

### Build any server
```bash
# From the golang directory; do not emit binaries into the repo
go build -o /dev/null ./cmd/<server>
```

### Run examples
```bash
go run ./examples/<server>
```

### Run a specific server
```bash
go build -o /tmp/fetch-mcp ./cmd/fetch-mcp
/tmp/fetch-mcp --ignore-robots-txt --proxy-url "http://proxy.example.com:8080"

go build -o /tmp/fs-mcp ./cmd/fs-mcp
/tmp/fs-mcp ~/code/project ~/notes

go build -o /tmp/memory-mcp ./cmd/memory-mcp
/tmp/memory-mcp --memory-file-path /tmp/mcp-memory.jsonl

DISABLE_THOUGHT_LOGGING=true go run ./cmd/sequentialthinking

go build -o /tmp/time-mcp ./cmd/time-mcp
/tmp/time-mcp

go build -o /tmp/git-mcp ./cmd/git-mcp
/tmp/git-mcp                              # unrestricted
/tmp/git-mcp --repository /path/to/repo   # restricted to one repo

go build -o /tmp/everything ./cmd/everything
/tmp/everything sse   # or: stdio | streamableHttp
```

### Testing
```bash
# Local tests (match project conventions)
go test -p=1 -count=1 -cover ./...

# Or via Docker helper
make dev-image
make unit-tests
```

### MCP Inspector
```bash
npx @modelcontextprotocol/inspector ./<built-binary>
```

## Server Highlights & Docs

| Server | Status | Primary Docs |
| --- | --- | --- |
| Everything | ✅ Implemented | [`cmd/everything/README.md`](./cmd/everything/README.md) · [`docs/parity.md#🌐-everything-server`](./docs/parity.md#🌐-everything-server) |
| Fetch | ✅ Implemented | [`cmd/fetch-mcp/README.md`](./cmd/fetch-mcp/README.md) · [`docs/parity.md#🌐-fetch-server`](./docs/parity.md#🌐-fetch-server) |
| Filesystem | ✅ Implemented | [`cmd/fs-mcp/README.md`](./cmd/fs-mcp/README.md) · [`docs/parity.md#📁-filesystem-server`](./docs/parity.md#📁-filesystem-server) |
| Memory | ✅ Implemented | [`cmd/memory-mcp/README.md`](./cmd/memory-mcp/README.md) · [`docs/parity.md#🧠-memory-server`](./docs/parity.md#🧠-memory-server) |
| Sequential Thinking | ✅ Implemented | [`cmd/sequentialthinking/README.md`](./cmd/sequentialthinking/README.md) · [`docs/parity.md#🧠-sequential-thinking-server`](./docs/parity.md#🧠-sequential-thinking-server) |
| Time | ✅ Implemented | [`cmd/time-mcp/README.md`](./cmd/time-mcp/README.md) · [`docs/parity.md#⏰-time-server`](./docs/parity.md#⏰-time-server) |
| Git | ✅ Implemented | [`cmd/git-mcp/README.md`](./cmd/git-mcp/README.md) · [`docs/parity.md#🗂️-git-server`](./docs/parity.md#🗂️-git-server) |

## Contributing
- Keep `docs/parity.md` updated when behavior diverges from upstream references.
- Favor stdlib and minimal dependencies; vendor changes with `go work vendor`.
- Follow project test conventions (`go test -p=1 -count=1 -cover ./...`), and document intentional gaps.
