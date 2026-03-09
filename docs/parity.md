# Fetch & Filesystem Server Parity Matrix

This document tracks feature parity between the Go implementations and their respective TypeScript/Python reference implementations.

## 📁 Filesystem Server

### ✅ Implemented Features

#### Core Operations
- **Read tools** - `read_text_file`, `read_media_file`, `read_multiple_files`
- **Write tools** - `create_directory`, `write_file`, `move_file` 
- **List & Search tools** - `list_directory`, `list_directory_with_sizes`, `search_files`, `directory_tree`
- **Edit tool** - `edit_file` with dry-run support via unified diff format
- **Metadata tools** - `get_file_info`, `list_allowed_directories`

#### Access Control
- Secure path validation preventing directory traversal
- Dynamic access control (support for reading roots to determine allowed paths)
- Handles absolute paths normalization accurately across OS platforms

### ⚠️ Known Differences & Limitations

#### Editing
- **TypeScript**: Uses highly advanced pattern matching, indentation preservation, and formatting heuristics for `edit_file`.
- **Go**: Uses simple string replacement for applying edits, and `sergi/go-diff` for dry-run preview. It doesn't attempt to format or preserve indentation in the same highly sophisticated way.

#### File Metadata
- **TypeScript**: Specifically retrieves advanced creation times and granular permissions.
- **Go**: Uses `os.Stat()`. Some granular timestamps (like `atime` and `ctime`) and permission representations might differ due to OS-agnostic behavior in standard Go packages.

#### Directory Tree Exclusions
- **TypeScript**: Uses custom logic or `picomatch`.
- **Go**: Uses `bmatcuk/doublestar/v4` for glob matching which covers all standard cases but may have slight edge-case differences in complex negative lookaheads.

---

## 🌐 Fetch Server

### ✅ Implemented Features

#### Core Functionality
- **Robots.txt enforcement** - Autonomous mode checks robots.txt before fetching; manual mode (prompts) bypasses checks
- **Dual user-agent support** - Different User-Agent strings for autonomous (`ModelContextProtocol/1.0 (Autonomous; ...)`) vs manual (`ModelContextProtocol/1.0 (User-Specified; ...)`) requests
- **HTML simplification** - Converts HTML to markdown using readability extraction + markdown conversion
- **Content chunking** - Supports `start_index` and `max_length` parameters with continuation hints when content is truncated
- **Raw mode** - Optional `raw` flag to skip HTML→markdown conversion and return original content
- **Proxy support** - Optional `--proxy-url` flag to route requests through a proxy
- **Tool exposure** - `fetch` tool registered with proper input schema validation
- **Prompt exposure** - `fetch` prompt registered for user-initiated fetches

#### Configuration Options
- `--user-agent` - Custom User-Agent string (overrides both autonomous and manual defaults)
- `--ignore-robots-txt` - Bypass robots.txt restrictions entirely
- `--proxy-url` - HTTP/HTTPS proxy URL for all requests

#### Error Handling
- URL validation and parsing errors
- HTTP connection failures with descriptive messages
- Robots.txt fetch failures (401/403 treated as blocking, 404 treated as allowing)
- Robots.txt parsing and enforcement with detailed error messages including robots.txt content
- Content extraction failures with fallback to raw content
- Proper MCP error responses with `IsError` flag set

### 🔄 Known Differences

#### HTML Processing
- **Python**: Uses `readabilipy` (Python library) for content extraction
- **Go**: Uses `go-shiori/go-readability` (Go port)
- **Impact**: Extraction algorithms may differ slightly, producing different simplified content for complex pages

#### Markdown Conversion
- **Python**: Uses `markdownify` library with ATX heading style
- **Go**: Uses `JohannesKaufmann/html-to-markdown` library
- **Impact**: Markdown formatting may differ (heading styles, list formatting, link formatting)

#### HTTP Client Behavior
- **Python**: Uses `httpx.AsyncClient` with async/await
- **Go**: Uses standard `net/http.Client` with context-based timeouts
- **Impact**: Redirect handling and timeout behavior may differ slightly

#### Robots.txt Parsing
- **Python**: Uses `protego` library
- **Go**: Uses `temoto/robotstxt` library
- **Impact**: Edge cases in robots.txt parsing may be handled differently

### ⚠️ Limitations

#### Not Implemented
- **Node.js fallback** - Python version mentions optional Node.js for enhanced HTML simplification; Go version does not have this fallback
- **Async architecture** - Python uses async/await; Go uses synchronous HTTP with context cancellation

#### Intentional Differences
- **Error message formatting** - Go error messages may be formatted differently than Python but convey the same information
- **Logging** - Go version uses MCP SDK's logging facilities; Python uses its own logging setup

## 📋 Testing Status

### Unit Tests ✅
- `robots_test.go` - Tests for robots.txt URL construction and comment processing
- `chunking_test.go` - Tests for content chunking logic and boundary conditions
- `path_validation_test.go` - Tests for secure path boundaries and sandbox enforcement.

### Integration Tests ⚠️
- Manual testing required with MCP inspector
- No automated integration tests yet (would require mock HTTP servers)

### Manual Testing Checklist
- [ ] Build binary: `go build -o server ./cmd/server_name`
- [ ] Test with MCP inspector: `npx @modelcontextprotocol/inspector ./server_name`

## 🎯 Validation Checklist

- [x] `go build ./cmd/...` succeeds
- [x] `go test ./core/pkg/tools/...` passes
- [x] Server registers with runtime without panic
- [x] Tool schema includes all required fields
- [x] Error handling uses proper MCP patterns (IsError flag)

---

## 🧠 Memory Server

### ✅ Implemented Features

#### Core Functionality
- **Graph Storage** - File-based storage (`memory.jsonl`) with entities, relations, and observations.
- **Migration** - Automatic fallback and migration from legacy `memory.json` formats to JSONL.
- **Tools Exposed** - `create_entities`, `create_relations`, `add_observations`, `delete_entities`, `delete_observations`, `delete_relations`, `read_graph`, `search_nodes`, `open_nodes`.

#### Configuration Options
- `MEMORY_FILE_PATH` - Override default storage location via environment variable.

### 🔄 Known Differences

#### Implementation Language
- **TypeScript**: Written in TypeScript using `fs.promises`. Uses Zod for schema validation.
- **Go**: Written in Go using standard `os` and `encoding/json` handling. Manual structural validation is enforced by strict Go structs and typed arguments.

#### Search
- **TypeScript**: Case-insensitive substring matching.
- **Go**: Case-insensitive substring matching using `strings.Contains`. Identical logic but implemented natively.

### ⚠️ Limitations

- Uses straightforward sequential file reads for JSONL which is identical to the TypeScript reference implementation, but might encounter identical scaling limits if the JSONL file grows exceptionally large.

## 📋 Testing Status
- Unit tests (`manager_test.go`, `server_test.go`) written via `testify/suite` exceeding 90% logic coverage for Graph Manager operations.

---

## 🧠 Sequential Thinking Server

### ✅ Implemented Features

#### Core Functionality
- **State Management** - In-memory history tracking and branch management for thoughts
- **Tool Exposure** - `sequentialthinking` tool registered with proper input schema validation matching TypeScript reference

#### Configuration Options
- `DISABLE_THOUGHT_LOGGING` - Environment variable to disable console output of thoughts

### 🔄 Known Differences

#### Implementation Language
- **TypeScript**: Written in TypeScript using standard string literal templates.
- **Go**: Written in Go using strings.Builder and formatted structs. Behavior matches identically.

### ⚠️ Limitations

- Operates purely in-memory. Terminating the MCP server process resets the thought history, matching the reference implementation.

## 📋 Testing Status
- Unit tests (`server_test.go`) written via `testify/suite`.

