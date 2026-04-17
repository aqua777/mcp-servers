# MCP Servers Parity Matrix

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

#### Output Format
- **Dual output modes** - All read/search tools support both `text` (human-readable) and `json` (structured) output
- **Server-level default** - `--output` / `-o` CLI flag sets the default format (`text` or `json`)
- **AI-first mode** - `--ai-mode` / `-a` CLI flag defaults to JSON output and enables structured error responses
- **Per-call override** - Each tool accepts an optional `format` parameter to override the server default

### 🆕 Go-Only Extensions (Not in TypeScript Reference)

#### Content Search: `grep`
A ripgrep-inspired tool for searching file contents by regex or literal pattern, not present in the TypeScript reference implementation.

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `path` | string | — | Directory or file to search (required) |
| `pattern` | string | — | Search pattern — regex or literal (required) |
| `fixedStrings` | bool | `false` | Treat pattern as literal string (`-F` in rg) |
| `ignoreCase` | bool | `false` | Case-insensitive match (`-i` in rg) |
| `smartCase` | bool | `false` | Case-insensitive unless pattern has uppercase (`-S` in rg) |
| `engine` | string | `re2` | Regex engine: `re2` (Go stdlib) or `pcre2` (lookahead/backreferences via `dlclark/regexp2`) |
| `includePatterns` | []string | — | Only search files matching these globs (`-g '*.ext'` in rg) |
| `excludePatterns` | []string | — | Skip files matching these globs (`-g '!*.ext'` in rg) |
| `contextBefore` | int | `0` | Lines of context before each match (`-B N` in rg) |
| `contextAfter` | int | `0` | Lines of context after each match (`-A N` in rg) |
| `contextLines` | int | `0` | Symmetric context lines (`-C N` in rg) |
| `maxMatches` | int | `1000` | Cap on total matches returned; `0` = unlimited |
| `format` | string | server default | Output format: `text` (ripgrep-style `path:line:text`) or `json` |

**JSON output** returns a structured `GrepResult` with `matches[]` (path, line number, line text, context before/after) and a `summary` (total matches, files matched, files searched, truncated flag, engine used).

**Binary detection**: files containing null bytes in the first 512 bytes are silently skipped, matching ripgrep's default behavior.

**Not in scope (future)**: multiline matching, word-boundary flag, invert match, count-only mode, file-type filters, max-depth, max-filesize, follow symlinks.

#### File Copy: `copy_file`

Copies files or directories. Not present in the TypeScript reference implementation.

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `source` | string | — | Source path or glob pattern (required) |
| `destination` | string | — | Destination path (required) |
| `recursive` | bool | `false` | Required to copy directories |
| `excludePatterns` | []string | — | Glob patterns to skip during recursive copy |
| `format` | string | server default | Output format: `text` or `json` |

**Glob mode**: when `source` contains `*`, `?`, or `[`, it is resolved via `doublestar.FilepathGlob` and all matches are copied into `destination` (treated as a directory). **Single-file mode**: plain source path; destination must not already exist (matches `move_file` semantics). Directories require `recursive: true`.

#### File Append: `append_file`

Appends content to a file, creating it if it does not exist. Not present in the TypeScript reference implementation.

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `path` | string | — | File to append to (required) |
| `content` | string | — | Content to append (required) |
| `format` | string | server default | Output format: `text` or `json` |

Uses `O_APPEND|O_CREATE|O_WRONLY`. Response includes `created: true` when the file was newly created.

#### Symbolic Link Creation: `create_symlink`

Creates a symbolic link. Not present in the TypeScript reference implementation.

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `target` | string | — | Path the symlink points to (required) |
| `path` | string | — | Path of the symlink to create (required) |
| `format` | string | server default | Output format: `text` or `json` |

Both `path` and `target` are validated against allowed directories to prevent sandbox escape via symlink. Dangling symlinks (target does not exist yet) are permitted — matches standard `ln -s` behaviour.

### 📋 Testing Status
- `grep_tools_test.go` — 30+ test cases covering all parameters, both engines, context lines, glob filters, binary detection, truncation, text and JSON formatters
- `read_tools_test.go`, `write_tools_test.go`, `list_tools_test.go`, `edit_tools_test.go`, `copy_append_symlink_tools_test.go` — full coverage of all other tools
- `path_validation_test.go` — sandbox enforcement
- `formatters_test.go` — formatter unit tests
- **Coverage**: ≥ 93% across the `filesystem` package

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

---

## ⏰ Time Server

### ✅ Implemented Features

#### Core Functionality
- **Time Retrieval** - Get current time with timezone, day of week, and DST status.
- **Timezone Conversion** - Convert a specific time (HH:MM) from a source timezone to a target timezone.
- **Tools Exposed** - `get_current_time`, `convert_time` registered with proper input schema validation matching TypeScript/Python reference.

### 🔄 Known Differences

#### Implementation Language
- **Python**: Uses `zoneinfo`, `tzlocal`, and standard `datetime`.
- **Go**: Uses standard `time` package with IANA timezone loading. It handles fractional hour offsets smoothly, consistent with Python reference.

### ⚠️ Limitations

- The Python reference defaults to the host machine's local timezone if none is provided via `tzlocal`. The Go version currently relies on the standard `time.Local`, though both require explicit timezone string inputs as per the MCP tool schema.

## 📋 Testing Status
- Unit tests (`server_test.go`) written via `testify/suite` achieving high coverage for timezone parsing, formatting, and mathematical offset conversions.

---

## 🌐 Everything Server

### ✅ Implemented Features

#### Core Functionality
- **Transports** - Standard stdio transport, plus SSE and Streamable HTTP support on port `3001` (using a server-per-session model).
- **Tools** - Implemented basic demo tools: `echo`, `get-env`, `get-sum`, `get-tiny-image`, `get-annotated-message`, `get-structured-content`.
- **Advanced Tools** - Implemented `get-roots-list`, `trigger-elicitation-request`, `trigger-sampling-request`, and `gzip-file-as-resource`.
- **Prompts** - `simple-prompt` and `args-prompt` available.
- **Resources** - Simulated resource template logic for text/blob URIs (`demo://resource/dynamic/text/{id}`).
- **Notifications** - `logging` background emitter testing `Level` propagation, plus `subscriptions` update emitter logic for resources.

### ⚠️ Known Differences & Limitations

#### Tools & Features
- **TypeScript**: Has 18 tools, including comprehensive tasks/research scenarios, async bi-directional elicitation requests, and complex completable prompts with state dependencies.
- **Go**: Implements a representative subset of tools. The advanced SEP-1686 task demos (e.g. `simulate-research-query` or bidirectional `trigger-sampling-request-async`) are currently omitted because the Go SDK does not yet support the experimental Tasks API (`experimental.tasks`) or the `TaskStore` required for these features.
- **Session Mapping**: Go SDK handles multiple sessions differently than the TS wrapper. For SSE and Streamable HTTP transports, we now use a server-per-session model similar to TS to properly track sessions.

## 📋 Testing Status
- Basic tools and server setup are tested via `go test`. Coverage includes error handling serialization formatting logic. Advanced tools like gzip, elicitation, and sampling have basic registration and null-session bounds checking.

---

## 🗂️ Git Server

### ✅ Implemented Features

#### Core Git Operations
- **`git_status`** - Working tree status
- **`git_diff_unstaged`** - Unstaged diff (working tree vs index)
- **`git_diff_staged`** - Staged diff (index vs HEAD)
- **`git_diff`** - Diff between current HEAD and a target branch/commit
- **`git_commit`** - Record staged changes with a message
- **`git_add`** - Stage specific files or all changes (`"."`)
- **`git_reset`** - Mixed reset to HEAD (unstage all)
- **`git_log`** - Commit history with optional max count and date range filtering
- **`git_create_branch`** - Create a branch from HEAD or an explicit base branch
- **`git_checkout`** - Switch to an existing branch
- **`git_show`** - Commit metadata and diff for any revision
- **`git_branch`** - List local/remote/all branches with optional `contains`/`not_contains` SHA filters

#### Output Format
- **Dual output modes** - All 7 read-only tools (`git_status`, `git_diff_unstaged`, `git_diff_staged`, `git_diff`, `git_log`, `git_show`, `git_branch`) support both text and JSON output
- **Server-level default** - `--output` / `-o` CLI flag sets the default format (`text` or `json`)
- **AI-first mode** - `--ai-mode` / `-a` CLI flag defaults to JSON output and enables structured error responses
- **Per-call override** - Each read tool accepts a `format` parameter to override the server default
- **Structured JSON** - JSON output includes rich metadata: repository info, file changes with line numbers, diff summaries, commit refs, branch tracking info
- **Git CLI text** - Text output matches `git` CLI formatting (e.g., `git log`, `git status`, unified diff)

#### Advanced Diff Features
- **`include_diff_content`** - Boolean parameter for diff operations (`git_diff_unstaged`, `git_diff_staged`, `git_diff`, `git_show`) to toggle line-level changes in JSON output (default: `true`). When `false`, returns file-level metadata only (path, status, additions, deletions) without the `changes[]` array
- **`max_files`** - Integer parameter for diff operations to limit the number of files included in the result (0 = unlimited). When truncated, the `truncated` field is set to `true` in the JSON response
- **Context after** - Diff changes now include `context_after` field with up to 3 lines of context following each addition/deletion, in addition to the existing `context_before`

#### File Filtering (Go-only Extension)

All file-listing tools (`git_status`, `git_diff_unstaged`, `git_diff_staged`, `git_diff`, `git_show`) accept three additional parameters for narrowing results:

| Parameter | Type | Default | Description |
|---|---|---|---|
| `include_patterns` | `string[]` | `[]` | Glob whitelist — only matching files appear |
| `exclude_patterns` | `string[]` | `[]` | Glob blacklist — matching files are removed |
| `no_gitignore` | `boolean` | `false` | Disable `.gitignore`-based filtering |

Filters are applied independently in order: gitignore → include → exclude. Glob matching uses `bmatcuk/doublestar/v4` (already vendored for the Filesystem server). Gitignore pattern loading uses `go-git/v5/plumbing/format/gitignore.ReadPatterns`.

**Not in Python reference** — the Python implementation returns all changed files without any path filtering.

#### Enhanced Metadata
- **Ahead/behind computation** - `git_status` now computes actual ahead/behind counts for tracked branches by walking the commit graph, replacing the hardcoded "up_to_date" status
- **Untracked file types** - `git_status` JSON output includes a `type` field ("file" or "directory") for untracked entries
- **Detached HEAD support** - `git_branch` detects detached HEAD state and sets `is_detached: true` with the short SHA as `current_branch`
- **Structured errors** - When using JSON format or AI mode, errors are returned as structured JSON with `error.code` and `error.message` fields

#### Security
- **Repository path restriction** - `--repository` / `GIT_REPOSITORY` flag restricts all `repo_path` arguments to a configured directory
- **Symlink-safe path validation** - Uses `filepath.EvalSymlinks` on both sides to prevent traversal via symlinks
- **Flag injection protection** - Ref names starting with `-` are rejected before any git operation

### 🔄 Known Differences

#### Git Library
- **Python**: Uses `GitPython` which shells out to the system `git` binary for many operations.
- **Go**: Uses `go-git/go-git/v5` — a **pure Go** reimplementation. No system `git` binary required.

#### Diff Output Format
- **Python**: `git_diff_unstaged` and `git_diff_staged` call `repo.git.diff()` which uses the system git binary's unified diff format directly.
- **Go**: Produces unified diff output using go-git's `UnifiedEncoder` and a custom character-level diff for unstaged changes. The output is functionally equivalent but may differ in whitespace or hunk headers in edge cases.

#### Structured Output
- **Python**: Returns plain text strings only.
- **Go**: Returns either Git CLI-style text or structured JSON with full metadata (file paths, line numbers, change types, summaries). JSON mode is an extension not present in the Python reference.

#### Commit Authorship
- **Python**: Uses the configured git identity from `.gitconfig`.
- **Go**: Defaults to `MCP Git Server <mcp@localhost>` since go-git does not read `.gitconfig` automatically.

#### Date Filtering in `git_log`
- **Python**: Passes ISO 8601 timestamps directly to GitPython.
- **Go**: Parses timestamps with support for ISO 8601, date-only (`YYYY-MM-DD`), and relative formats (`2 weeks ago`, `yesterday`, `3 days ago`, `1 month ago`, etc.)

### ⚠️ Limitations

#### Not Implemented
- **Remote operations** — `git_push`, `git_pull`, `git_fetch` are not part of the Python reference tool set and are not implemented.
- **Stash** — Not in the Python reference.
- **Merge/Rebase** — Not in the Python reference.

#### Intentional Differences
- **Commit author** — Go version uses a fixed author signature; Python uses system git identity.
- **Bare repository support** — Operations requiring a worktree (status, diff, add, commit, reset, checkout) will return an error on bare repos. This matches practical usage.

## 📋 Testing Status
- Unit tests (`validation_test.go`, `server_test.go`, `filter_test.go`) written via `testify/suite`.
- Coverage: **88.9%** — slightly below the 90% threshold due to pre-existing hard-to-test paths (see exceptions below).
- `filter_test.go` — 18 test cases covering `FileFilter`, `filterDiffFiles`, gitignore matching, include/exclude glob patterns, combined filters, edge cases
- Coverage exceptions (documented per AGENTS.md policy):
  - `computeAheadBehind` — requires a real remote repository with push/fetch history (~17 statements)
  - Remote-tracking branch paths in `gitStatus` — require a configured remote ref
  - Merge conflict status path — requires a partially-applied merge (not creatable via go-git's high-level API)
  - Worktree/NewFileFilter error paths in handlers — require filesystem-level injection not possible without a custom `billy.Filesystem`
  - `filter.go` gitignore ReadPatterns error path — requires a billy.Filesystem that errors mid-traversal
