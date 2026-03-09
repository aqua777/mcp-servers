# Experimental Go MCP Servers

> ⚠️ **Caution**: This repository contains AI-transposed, experimental ports of the reference MCP servers from `servers/src`. Every binary is under active construction—APIs may change, transports may be unstable, and parity with the upstream TypeScript/Python implementations is still being validated. Use at your own risk and do not deploy in production.

## Goals
- Provide Go-native equivalents of the Everything, Filesystem, Fetch, Git, Memory, Sequential Thinking, and Time reference servers.
- Expose each server as both a standalone binary (`cmd/<server>`) and an importable Go package for embedding.
- Track feature parity exhaustively so gaps are visible and testable.
- Ship ergonomic example programs in `./examples/<server>` for every implemented server.

## Workspace Layout
```
./go.work                # References all modules (core + per-server)
./core                   # Shared runtime, config, logging, and registration helpers
./cmd/<server>           # Individual binaries with their own go.mod
./docs/parity.md         # Snapshot of upstream parity + TODOs
./examples/<server>      # Demonstrations of how to embed the servers
```

## Implementation Status

### ✅ Completed
- **Fetch** – See [`cmd/fetch/README.md`](./cmd/fetch/README.md) for CLI + runtime wiring details and [`examples/fetch`](./examples/fetch) for embedding usage.
- **Filesystem** – See [`docs/parity.md#📁-filesystem-server`](./docs/parity.md#📁-filesystem-server) and [`examples/filesystem`](./examples/filesystem) for the demo runner.
- **Memory** – See [`cmd/memory/README.md`](./cmd/memory/README.md) plus [`examples/memory`](./examples/memory) for SDK usage.
- **Sequential Thinking** – See [`cmd/sequentialthinking/README.md`](./cmd/sequentialthinking/README.md) for tooling details and [`examples/sequentialthinking`](./examples/sequentialthinking) for embedding usage. Parity notes live in [`docs/parity.md#🧠-sequential-thinking-server`](./docs/parity.md#🧠-sequential-thinking-server).
- **Time** – See [`cmd/time/README.md`](./cmd/time/README.md) for CLI details and [`examples/time`](./examples/time) for SDK usage. Parity notes live in [`docs/parity.md#⏰-time-server`](./docs/parity.md#⏰-time-server).

### 🚧 In Progress / Planned
- **Everything** – MCP runtime placeholder
- **Git** – Placeholder package

## Current Status (high-level)
- ✅ Workspace + module scaffolding in place
- ✅ Runtime glue + server registry system implemented
- ✅ Fetch, Filesystem, Memory, and Sequential Thinking servers implemented with docs + examples
- 🚧 Remaining servers are placeholders awaiting implementation

## Getting Started

### Building the Fetch Server
```bash
# From the golang directory
go build -o fetch ./cmd/fetch

# Run with default settings
./fetch

# Run with custom options
./fetch --user-agent "MyBot/1.0" --ignore-robots-txt --proxy-url "http://proxy.example.com:8080"
```

### Building the Filesystem Server
```bash
# From the golang directory
go build -o filesystem ./cmd/filesystem

# Allow the server to manage two directories
./filesystem ~/code/project ~/notes
```

### Building the Memory Server
```bash
# From the golang directory
go build -o memory ./cmd/memory

# Point at a custom JSONL store
./memory --memory-file-path /tmp/mcp-memory.jsonl
```

### Building the Sequential Thinking Server
```bash
# From the golang directory
go build -o sequentialthinking ./cmd/sequentialthinking

# Run with thought logging disabled
DISABLE_THOUGHT_LOGGING=true ./sequentialthinking
```

### Testing
```bash
# Run all fetch server tests
go test ./core/pkg/tools/fetch -v

# Run filesystem tests (serialized due to file operations)
go test -p=1 ./core/pkg/tools/filesystem -v

# Run memory tests (suite-based)
go test ./core/pkg/tools/memory -v

# Run sequential thinking tests
go test ./core/pkg/tools/sequentialthinking -v
```

### Using with MCP Inspector
```bash
# Install MCP inspector (if not already installed)
npm install -g @modelcontextprotocol/inspector

# Run the fetch server with inspector
npx @modelcontextprotocol/inspector ./fetch
```

## Server Highlights & Docs

| Server | Status | Primary Docs |
| --- | --- | --- |
| Fetch | ✅ Implemented | [`cmd/fetch/README.md`](./cmd/fetch/README.md) · [`docs/parity.md#🌐-fetch-server`](./docs/parity.md#🌐-fetch-server) |
| Filesystem | ✅ Implemented | [`docs/parity.md#📁-filesystem-server`](./docs/parity.md#📁-filesystem-server) |
| Memory | ✅ Implemented | [`cmd/memory/README.md`](./cmd/memory/README.md) · [`docs/parity.md#🧠-memory-server`](./docs/parity.md#🧠-memory-server) |
| Sequential Thinking | ✅ Implemented | [`cmd/sequentialthinking/README.md`](./cmd/sequentialthinking/README.md) · [`docs/parity.md#🧠-sequential-thinking-server`](./docs/parity.md#🧠-sequential-thinking-server) |
| Time | ✅ Implemented | [`cmd/time/README.md`](./cmd/time/README.md) · [`docs/parity.md#⏰-time-server`](./docs/parity.md#⏰-time-server) |
| Everything | 🚧 Placeholder | _TBD_ |
| Git | 🚧 Placeholder | _TBD_ |

## Contributing
- Review `docs/parity.md` for the outstanding parity tasks per server.
- Prefer minimal, well-reviewed third-party dependencies; stick to stdlib where practical.
- Document any divergence from the upstream reference behavior directly in the parity doc.
