package filesystem

// DirectoryEntry represents a single file or directory entry.
type DirectoryEntry struct {
	Name string `json:"name"`
	Type string `json:"type"` // "file" or "directory"
	Size *int64 `json:"size,omitempty"`
}

// DirectorySummary provides aggregate counts.
type DirectorySummary struct {
	Files       int   `json:"files"`
	Directories int   `json:"directories"`
	TotalSize   int64 `json:"total_size,omitempty"`
}

// ListDirectoryResult holds the structured result of a list_directory operation.
type ListDirectoryResult struct {
	Path    string           `json:"path"`
	Entries []DirectoryEntry `json:"entries"`
	Summary DirectorySummary `json:"summary"`
	SortBy  string           `json:"sort_by,omitempty"`
}

// Node represents a directory tree node (already exists in list_tools.go).
// DirectoryTreeResult wraps it with summary.
type DirectoryTreeResult struct {
	Root    *Node            `json:"root"`
	Summary DirectorySummary `json:"summary"`
}

// SearchMatch represents a single search result.
type SearchMatch struct {
	Path string `json:"path"`
	Type string `json:"type"`
}

// SearchSummary provides search result counts.
type SearchSummary struct {
	TotalMatches int  `json:"total_matches"`
	Truncated    bool `json:"truncated,omitempty"`
}

// SearchResult holds the structured result of a search_files operation.
type SearchResult struct {
	Root    string        `json:"root"`
	Pattern string        `json:"pattern"`
	Matches []SearchMatch `json:"matches"`
	Summary SearchSummary `json:"summary"`
}

// ReadFileResult holds the structured result of a read_text_file operation.
type ReadFileResult struct {
	Path    string `json:"path"`
	Size    int64  `json:"size"`
	Lines   int    `json:"lines"`
	Content string `json:"content"`
}

// FileReadResult represents a single file read in read_multiple_files.
type FileReadResult struct {
	Path    string `json:"path"`
	Status  string `json:"status"` // "ok" or "error"
	Content string `json:"content,omitempty"`
	Error   string `json:"error,omitempty"`
}

// ReadMultipleFilesSummary provides counts.
type ReadMultipleFilesSummary struct {
	Succeeded int `json:"succeeded"`
	Failed    int `json:"failed"`
	Total     int `json:"total"`
}

// ReadMultipleFilesResult holds the structured result of a read_multiple_files operation.
type ReadMultipleFilesResult struct {
	Files   []FileReadResult         `json:"files"`
	Summary ReadMultipleFilesSummary `json:"summary"`
}

// EditFileResult holds the structured result of an edit_file operation.
type EditFileResult struct {
	Path           string `json:"path"`
	Status         string `json:"status"` // "dry_run" or "applied"
	Diff           string `json:"diff,omitempty"`
	EditsApplied   int    `json:"edits_applied"`
	EditsRequested int    `json:"edits_requested"`
}

// FileInfoResult holds the structured result of a get_file_info operation.
type FileInfoResult struct {
	Path         string `json:"path"`
	Name         string `json:"name"`
	Size         int64  `json:"size"`
	Type         string `json:"type"`
	Permissions  string `json:"permissions"`
	ModifiedTime string `json:"modified_time"`
	CreatedTime  string `json:"created_time,omitempty"`
	AccessedTime string `json:"accessed_time,omitempty"`
	IsSymlink    bool   `json:"is_symlink"`
}

// WriteResult holds the structured result of a write_file operation.
type WriteResult struct {
	Path         string `json:"path"`
	Status       string `json:"status"`
	BytesWritten int    `json:"bytes_written"`
}

// CreateDirectoryResult holds the structured result of a create_directory operation.
type CreateDirectoryResult struct {
	Path    string `json:"path"`
	Status  string `json:"status"`
	Created bool   `json:"created"` // false if already existed
}

// MoveResult holds the structured result of a move_file operation.
type MoveResult struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
	Status      string `json:"status"`
}

// AllowedDirectoriesResult holds the structured result of a list_allowed_directories operation.
type AllowedDirectoriesResult struct {
	AllowedDirectories []string `json:"allowed_directories"`
}

// ContextLine is a non-matching line included for context around a match.
type ContextLine struct {
	LineNumber int    `json:"line_number"`
	LineText   string `json:"line_text"`
}

// GrepMatch represents a single matching line within a file.
type GrepMatch struct {
	Path          string        `json:"path"`
	LineNumber    int           `json:"line_number"`
	LineText      string        `json:"line_text"`
	ContextBefore []ContextLine `json:"context_before,omitempty"`
	ContextAfter  []ContextLine `json:"context_after,omitempty"`
}

// GrepSummary provides aggregate counts for a grep operation.
type GrepSummary struct {
	TotalMatches  int    `json:"total_matches"`
	FilesMatched  int    `json:"files_matched"`
	FilesSearched int    `json:"files_searched"`
	Truncated     bool   `json:"truncated,omitempty"`
	Engine        string `json:"engine"`
}

// GrepResult holds the structured result of a grep operation.
type GrepResult struct {
	Pattern string      `json:"pattern"`
	Root    string      `json:"root"`
	Matches []GrepMatch `json:"matches"`
	Summary GrepSummary `json:"summary"`
}

// ErrorResponse provides structured error output in JSON mode.
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail holds error metadata.
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
