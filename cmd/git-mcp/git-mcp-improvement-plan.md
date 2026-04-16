# Git-MCP Server Improvement Plan for AI Consumption

## Problem Statement

The git-mcp server currently provides Git read operations that return human-readable text formats, which are suboptimal for AI coding assistants. The outputs require complex parsing, lose critical metadata, and are inconsistent across operations. This document defines the improvements needed to make git-mcp suitable for AI consumption.

## Scope

This plan covers 7 Git read operations:
1. `git_branch` - List branches
2. `git_diff` - Compare commits
3. `git_diff_staged` - Show staged changes
4. `git_diff_unstaged` - Show unstaged changes
5. `git_log` - Show commit history
6. `git_show` - Show commit details
7. `git_status` - Show working tree status

## Critical Issues (Priority 1)

### Issue 1: Loss of Critical Metadata

**Affected Operations:** `git_log`, `git_show`, `git_status`

**Problem:** Current implementation strips essential context that AI needs for repository analysis:
- Author email addresses (only name provided)
- Branch references (HEAD, origin refs, remote tracking)
- Remote sync status (ahead/behind counts)
- Merge parent information
- File status types (renamed, deleted, copied - only shows modified)

**Impact:** AI cannot determine:
- Who committed changes (email needed for attribution)
- Which branches contain commits
- Repository synchronization state
- Merge relationships and history
- Complete file change context

**Required Fix:** Include all metadata from Git CLI output without loss.

---

### Issue 2: No Structured Data Format

**Affected Operations:** All 7 operations

**Problem:** All outputs are unstructured text requiring:
- Regex parsing for extraction
- State machine logic for diff parsing
- Heuristics to determine current state
- Error-prone edge case handling

**Impact:** AI must implement complex parsing logic that is fragile and maintenance-heavy.

**Required Fix:** Provide structured JSON output format for all operations.

---

### Issue 3: Inconsistent Empty State Handling

**Affected Operations:** `git_diff`, `git_diff_staged`

**Problem:** Different headers for same semantic state (no changes):
- `git_diff` returns "Diff with HEAD:" when empty
- `git_diff_staged` returns "Staged changes:" when empty

**Impact:** AI parsing logic must handle multiple formats for "no changes" state.

**Required Fix:** Return consistent empty state representation across all diff operations.

---

## High-Priority Issues (Priority 2)

### Issue 4: Missing Repository Context in git_status

**Affected Operations:** `git_status`

**Problem:** Current output lacks:
- Current branch name
- Remote sync status
- Remote branch name
- Separation of staged/unstaged/untracked files
- Conflict status

**Impact:** AI cannot assess repository state or plan next actions without context.

**Required Fix:** Include repository metadata and proper file categorization.

---

### Issue 5: Non-Standard Formatting

**Affected Operations:** `git_log`, `git_show`, `git_diff_unstaged`

**Problem:** Custom formats that don't match Git CLI:
- "Commit: SHA" instead of "commit SHA (refs)"
- Added headers like "Unstaged changes:" not in CLI
- Missing standard separators between log entries

**Impact:** AI cannot leverage existing Git parsing libraries or patterns.

**Required Fix:** Match Git CLI output format exactly when in text mode.

---

## Medium-Priority Issues (Priority 3)

### Issue 6: Diff Format Requires Complex Parsing

**Affected Operations:** `git_diff`, `git_diff_staged`, `git_diff_unstaged`, `git_show`

**Problem:** Unified diff format is human-readable but requires:
- Parsing diff headers
- Tracking hunk positions
- Extracting line numbers
- Handling binary files

**Impact:** High complexity for AI to extract meaningful change information.

**Required Fix:** Provide structured diff format with file-level and line-level change metadata.

---

### Issue 7: Branch Listing Lacks Metadata

**Affected Operations:** `git_branch`

**Problem:** Only provides branch names with `*` prefix for current branch:
- No remote tracking information
- No last commit metadata
- No branch relationship information

**Impact:** AI cannot determine branch status or relationships.

**Required Fix:** Include branch metadata (tracking, last commit, is_head, etc.).

---

## Improvement Roadmap

### Phase 1: Immediate Fixes (1-2 days)

**Goal:** Stop losing information, match Git CLI behavior

**Actions:**
1. Remove custom headers from all operations
2. Include all metadata from Git CLI (emails, refs, merge info)
3. Return empty string for empty states (no custom headers)
4. Match Git CLI output format exactly for text mode

**Acceptance Criteria:**
- `git_log` includes author email, branch refs, merge parents
- `git_show` includes all metadata from `git show` CLI
- `git_status` includes branch and remote information
- Empty states return empty string consistently
- All text outputs match Git CLI exactly

---

### Phase 2: Add JSON Format Option (3-5 days)

**Goal:** Provide structured output option for AI consumption

**Actions:**
1. Add optional `format` parameter to all operations (default: "text")
2. Implement JSON output for each operation
3. Maintain backward compatibility (text format unchanged)
4. Add validation for format parameter

**Parameter Specification:**
```typescript
interface GitOperationParams {
  format?: "text" | "json";  // Default: "text"
  // ... existing parameters
}
```

**JSON Schema Specifications:**

#### git_branch JSON Schema
```json
{
  "current_branch": "string",
  "branches": [
    {
      "name": "string",
      "is_current": boolean,
      "tracking": "string | null",
      "last_commit_sha": "string",
      "last_commit_date": "ISO8601"
    }
  ]
}
```

#### git_log JSON Schema
```json
{
  "commits": [
    {
      "sha": "string",
      "author": {
        "name": "string",
        "email": "string"
      },
      "date": "ISO8601",
      "message": "string",
      "refs": ["string"],
      "parents": ["string"]
    }
  ]
}
```

#### git_status JSON Schema
```json
{
  "repository": {
    "branch": "string",
    "head_sha": "string",
    "remote": {
      "name": "string",
      "branch": "string",
      "status": "up_to_date | ahead | behind | diverged",
      "ahead_by": number,
      "behind_by": number
    }
  },
  "changes": {
    "staged": [
      {
        "path": "string",
        "status": "modified | added | deleted | renamed | copied"
      }
    ],
    "unstaged": [/* same structure */],
    "untracked": [{"path": "string", "type": "file | directory"}],
    "conflicts": [{"path": "string"}]
  },
  "summary": {
    "total_files": number,
    "staged_count": number,
    "unstaged_count": number,
    "untracked_count": number,
    "conflicted_count": number
  }
}
```

#### git_diff (all variants) JSON Schema
```json
{
  "status": "has_changes | no_changes",
  "base": "string",
  "target": "string",
  "files": [
    {
      "path": "string",
      "status": "modified | added | deleted | renamed",
      "additions": number,
      "deletions": number,
      "changes": [
        {
          "type": "line_change | hunk_header",
          "old_line": number,
          "new_line": number,
          "old_content": "string",
          "new_content": "string"
        }
      ]
    }
  ]
}
```

#### git_show JSON Schema
```json
{
  "commit": {
    "sha": "string",
    "author": {"name": "string", "email": "string"},
    "date": "ISO8601",
    "message": "string",
    "refs": ["string"],
    "parents": ["string"]
  },
  "diff": {
    "files": [/* same as git_diff files structure */]
  }
}
```

**Acceptance Criteria:**
- All 7 operations accept `format` parameter
- JSON output validates against schemas
- Text output unchanged (backward compatible)
- JSON includes all metadata from text output
- Empty states handled consistently in JSON

---

### Phase 3: Enhanced Diff Structure (5-7 days)

**Goal:** Provide machine-readable diff format

**Actions:**
1. Implement structured diff parsing for JSON mode
2. Extract line-level changes with positions
3. Calculate addition/deletion counts per file
4. Handle binary files appropriately
5. Support rename/copy detection

**Enhanced Diff Schema:**
```json
{
  "files": [
    {
      "path": "string",
      "old_path": "string | null",  // For renames
      "status": "modified | added | deleted | renamed | copied",
      "binary": boolean,
      "additions": number,
      "deletions": number,
      "changes": [
        {
          "type": "addition | deletion | modification",
          "old_line": number,
          "new_line": number,
          "old_content": "string",
          "new_content": "string",
          "context_before": ["string"],
          "context_after": ["string"]
        }
      ]
    }
  ]
}
```

**Acceptance Criteria:**
- Line-level changes extracted with accurate positions
- Addition/deletion counts accurate
- Binary files flagged correctly
- Rename/copy operations detected
- Context lines included for changes

---

### Phase 4: AI-First Defaults (Future)

**Goal:** Optimize default behavior for AI consumption

**Actions:**
1. Change default format to "json"
2. Add AI-specific options (e.g., `include_diff_content: boolean`)
3. Provide change summaries by default
4. Add operation-specific optimizations

**Considerations:**
- Breaking change - requires major version bump
- Maintain text mode for human use cases
- Document AI-optimized usage patterns

---

## Implementation Guidelines

### Error Handling

All operations should return structured error responses in JSON mode:

```json
{
  "error": {
    "code": "string",
    "message": "string",
    "details": {}
  }
}
```

### Backward Compatibility

- Text format must remain unchanged
- JSON format is opt-in via parameter
- Default behavior (no format param) = text mode
- Document deprecation timeline if changing defaults

### Performance Considerations

- JSON parsing overhead should be minimal
- Consider caching parsed results for repeated calls
- Large diffs may need streaming or pagination
- Provide options to limit output size (e.g., `max_files`, `max_commits`)

### Testing Strategy

For each operation:
1. Unit tests for JSON schema validation
2. Integration tests comparing JSON vs text output
3. Edge case tests (empty repo, large diffs, merge conflicts)
4. Regression tests to ensure text format unchanged

---

## Success Metrics

### Phase 1 Success
- Zero information loss vs Git CLI
- AI can parse text output with same complexity as Git CLI
- All metadata present in outputs

### Phase 2 Success
- AI can consume JSON output without regex parsing
- JSON output contains all information from text output
- Backward compatibility maintained (text unchanged)

### Phase 3 Success
- AI can extract line-level changes without diff parsing
- Change statistics accurate and complete
- Binary files handled correctly

### Overall Success
- AI assistants prefer git-mcp over direct Git CLI calls
- Reduced complexity in AI parsing logic
- Improved accuracy in repository analysis
- Better user experience for AI-powered workflows

---

## Open Questions

1. **Default Format:** Should we eventually switch default to JSON? (Requires major version)
2. **Diff Granularity:** Should line-level changes be included by default or opt-in?
3. **Large Repositories:** How to handle diffs with thousands of files? Pagination? Streaming?
4. **Binary Files:** Should binary diffs include file size/hash information?
5. **Merge Conflicts:** Should git_status include conflict markers/sections in JSON?

---

## Dependencies

- Git CLI version compatibility
- JSON schema validation library
- Diff parsing library (consider libgit2 or go-git)
- Testing framework for JSON schema validation

---

## Risks and Mitigations

**Risk:** Breaking changes to existing integrations
**Mitigation:** Maintain text format as default, JSON as opt-in

**Risk:** Performance degradation from JSON generation
**Mitigation:** Benchmark and optimize, provide options to limit output

**Risk:** Schema evolution causing compatibility issues
**Mitigation:** Version JSON schemas, document changes clearly

**Risk:** Complex diff parsing introduces bugs
**Mitigation:** Comprehensive testing, leverage existing libraries

---

## Appendix A: Operation-Specific Requirements

### git_branch
- MUST include remote tracking branch if set
- MUST identify current branch explicitly
- SHOULD include last commit SHA and date
- SHOULD handle detached HEAD state

### git_log
- MUST include author name and email
- MUST include branch references (HEAD, origin refs)
- MUST include merge parent SHAs for merge commits
- MUST preserve multi-line commit message formatting
- SHOULD support pagination for long histories

### git_show
- MUST include all git_log metadata
- MUST include full diff in structured format
- SHOULD handle merge commits (show merge summary)

### git_status
- MUST include current branch name
- MUST include remote sync status
- MUST categorize files (staged/unstaged/untracked/conflicts)
- MUST identify file status type (modified/added/deleted/renamed)
- SHOULD include ahead/behind counts

### git_diff (all variants)
- MUST identify base and target refs/SHAs
- MUST handle empty state consistently
- MUST categorize file changes
- SHOULD include addition/deletion counts
- SHOULD handle binary files appropriately

---

## Appendix B: Example Usage

### Before (Current)
```typescript
// AI must parse this:
const output = await git_log({ repo_path: "/path", max_count: 5 });
// Returns:
// Commit history:
// Commit: "1c432c94794e6424907fa01b0c4f187c51e4e218"
// Author: "Piotr Gridniew"
// Date: 2026-04-16T09:51:56+01:00
// Message: "PI-45586: updating Makefile + README.md files"
// ... parsing required ...
```

### After (Phase 2)
```typescript
const output = await git_log({ 
  repo_path: "/path", 
  max_count: 5,
  format: "json" 
});
// Returns:
{
  "commits": [
    {
      "sha": "1c432c94794e6424907fa01b0c4f187c51e4e218",
      "author": {
        "name": "Piotr Gridniew",
        "email": "piotrg@qualtrics.com"
      },
      "date": "2026-04-16T09:51:56+01:00",
      "message": "PI-45586: updating Makefile + README.md files",
      "refs": ["HEAD", "piotrg-PI-45586-04", "origin/piotrg-PI-45586-04"],
      "parents": ["d87852b42553c3bf499640afcf80f64186c3c329"]
    }
  ]
}
// No parsing required - direct property access
```

### After (Phase 3 - Enhanced Diff)
```typescript
const output = await git_diff_unstaged({ 
  repo_path: "/path",
  format: "json" 
});
// Returns:
{
  "status": "has_changes",
  "files": [
    {
      "path": "plugins/k8s-consul/config.go",
      "status": "modified",
      "additions": 16,
      "deletions": 0,
      "changes": [
        {
          "type": "addition",
          "old_line": 10,
          "new_line": 10,
          "new_content": "TTL uint32",
          "context_before": ["type PluginConfig struct {"],
          "context_after": ["}"]
        }
      ]
    }
  ]
}
// AI can directly analyze changes without diff parsing
```
