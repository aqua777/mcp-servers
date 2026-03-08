# Experimental Go MCP Servers

> ⚠️ **Caution**: This repository contains AI-transposed, experimental ports of the reference MCP servers from `servers/src`. Every binary is under active construction—APIs may change, transports may be unstable, and parity with the upstream TypeScript/Python implementations is still being validated. Use at your own risk and do not deploy in production.

## Goals
- Provide Go-native equivalents of the Everything, Filesystem, Git, Memory, Sequential Thinking, Time, and Fetch reference servers.
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
- **Fetch**
  - Robots.txt enforcement with autonomous/manual modes
  - Readability extraction + HTML→Markdown conversion (raw passthrough supported)
  - Content chunking with continuation hints
  - Proxy + custom user-agent configuration
- **Filesystem**
  - Read tools (`read_text_file`, `read_media_file`, `read_multiple_files`)
  - Write/edit tools (`write_file`, `move_file`, `create_directory`, `edit_file` w/ dry-run)
  - Discovery tools (`list_directory`, `directory_tree`, `search_files`, `list_allowed_directories`)
  - Secure path sandboxing + absolute path normalization across OSes

### 🚧 In Progress / Planned
- **Everything** - MCP runtime placeholder
- **Git** - Placeholder package
- **Memory** - Placeholder package
- **Sequential Thinking** - Placeholder package
- **Time** - Placeholder package

## Current Status (high-level)
- ✅ Workspace + module scaffolding in place
- ✅ Runtime glue + server registry system implemented
- ✅ Fetch server fully implemented with parity-tracked docs + examples
- ✅ Filesystem server implemented with parity docs, example app, and sandbox validations
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

### Testing
```bash
# Run all fetch server tests
go test ./core/pkg/tools/fetch -v

# Run filesystem tests (serialized due to file operations)
go test -p=1 ./core/pkg/tools/filesystem -v
```

### Using with MCP Inspector
```bash
# Install MCP inspector (if not already installed)
npm install -g @modelcontextprotocol/inspector

# Run the fetch server with inspector
npx @modelcontextprotocol/inspector ./fetch
```

## Server Highlights

### Fetch

Provides web content fetching with MCP tool + prompt exposure:

- **Tool** `fetch`
  - `url` (required)
  - `max_length` (optional, default 5000)
  - `start_index` (optional, default 0)
  - `raw` (optional, default false)
- **Prompt** `fetch`
  - `url` (required, bypasses robots.txt)
- **Flags**
  - `--user-agent`, `--ignore-robots-txt`, `--proxy-url`

See `docs/parity.md#🌐-fetch-server` for deeper feature parity notes.

### Filesystem

Exposes safe file management utilities rooted to whitelisted directories:

- **Tools**
  - Read: `read_text_file`, `read_media_file`, `read_multiple_files`
  - Write/Edit: `write_file`, `move_file`, `create_directory`, `edit_file`
  - Discovery: `list_directory`, `list_directory_with_sizes`, `directory_tree`, `search_files`
  - Metadata: `get_file_info`, `list_allowed_directories`
- **Sandboxing**
  - Absolute path validation + traversal prevention
  - Optional dry-run editing via diff previews

Parity considerations live in `docs/parity.md#📁-filesystem-server`.

## Contributing
- Review `docs/parity.md` for the outstanding parity tasks per server.
- Prefer minimal, well-reviewed third-party dependencies; stick to stdlib where practical.
- Document any divergence from the upstream reference behavior directly in the parity doc.
