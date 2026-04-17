package filesystem

import (
	"encoding/json"
	"fmt"
	"strings"
)

// formatListDirectoryText renders a ListDirectoryResult in text format.
func formatListDirectoryText(r *ListDirectoryResult) string {
	var lines []string
	for _, entry := range r.Entries {
		prefix := "[FILE]"
		if entry.Type == "directory" {
			prefix = "[DIR]"
		}
		if entry.Size != nil {
			lines = append(lines, fmt.Sprintf("%s %s (%d bytes)", prefix, entry.Name, *entry.Size))
		} else {
			lines = append(lines, fmt.Sprintf("%s %s", prefix, entry.Name))
		}
	}

	// Add summary if sizes are included
	if r.Summary.TotalSize > 0 || r.SortBy != "" {
		lines = append(lines, fmt.Sprintf("\nSummary: %d files, %d directories, %d bytes total",
			r.Summary.Files, r.Summary.Directories, r.Summary.TotalSize))
	}

	return strings.Join(lines, "\n")
}

// formatListDirectoryJSON renders a ListDirectoryResult as JSON.
func formatListDirectoryJSON(r *ListDirectoryResult) string {
	b, _ := json.Marshal(r)
	return string(b)
}

// formatDirectoryTreeText renders a DirectoryTreeResult in JSON format (tree is already JSON-native).
func formatDirectoryTreeText(r *DirectoryTreeResult) string {
	// Tree is inherently JSON; text mode returns JSON too
	var children []*Node
	if r.Root != nil && r.Root.Type == "directory" {
		children = r.Root.Children
	} else if r.Root != nil {
		children = []*Node{r.Root}
	}
	b, _ := json.MarshalIndent(children, "", "  ")
	return string(b)
}

// formatDirectoryTreeJSON renders a DirectoryTreeResult as JSON with summary.
func formatDirectoryTreeJSON(r *DirectoryTreeResult) string {
	b, _ := json.Marshal(r)
	return string(b)
}

// formatSearchFilesText renders a SearchResult in text format (newline-delimited paths).
func formatSearchFilesText(r *SearchResult) string {
	var paths []string
	for _, match := range r.Matches {
		paths = append(paths, match.Path)
	}
	return strings.Join(paths, "\n")
}

// formatSearchFilesJSON renders a SearchResult as JSON.
func formatSearchFilesJSON(r *SearchResult) string {
	b, _ := json.Marshal(r)
	return string(b)
}

// formatReadFileText renders a ReadFileResult in text format (raw content).
func formatReadFileText(r *ReadFileResult) string {
	return r.Content
}

// formatReadFileJSON renders a ReadFileResult as JSON.
func formatReadFileJSON(r *ReadFileResult) string {
	b, _ := json.Marshal(r)
	return string(b)
}

// formatReadMultipleFilesText renders a ReadMultipleFilesResult in text format.
func formatReadMultipleFilesText(r *ReadMultipleFilesResult) string {
	var results []string
	for _, file := range r.Files {
		if file.Status == "ok" {
			results = append(results, fmt.Sprintf("--- %s ---\n%s", file.Path, file.Content))
		} else {
			results = append(results, fmt.Sprintf("%s: Error: %s", file.Path, file.Error))
		}
	}
	return strings.Join(results, "\n\n")
}

// formatReadMultipleFilesJSON renders a ReadMultipleFilesResult as JSON.
func formatReadMultipleFilesJSON(r *ReadMultipleFilesResult) string {
	b, _ := json.Marshal(r)
	return string(b)
}

// formatEditFileText renders an EditFileResult in text format.
func formatEditFileText(r *EditFileResult) string {
	if r.Status == "dry_run" {
		return r.Diff
	}
	return fmt.Sprintf("Successfully edited %s", r.Path)
}

// formatEditFileJSON renders an EditFileResult as JSON.
func formatEditFileJSON(r *EditFileResult) string {
	b, _ := json.Marshal(r)
	return string(b)
}

// formatFileInfoText renders a FileInfoResult in JSON format (already JSON-native).
func formatFileInfoText(r *FileInfoResult) string {
	// Current implementation returns JSON even in text mode for backward compat
	result := map[string]any{
		"size":          r.Size,
		"modified_time": r.ModifiedTime,
		"type":          r.Type,
		"permissions":   r.Permissions,
	}
	b, _ := json.MarshalIndent(result, "", "  ")
	return string(b)
}

// formatFileInfoJSON renders a FileInfoResult as JSON.
func formatFileInfoJSON(r *FileInfoResult) string {
	b, _ := json.Marshal(r)
	return string(b)
}

// formatWriteFileText renders a WriteResult in text format.
func formatWriteFileText(r *WriteResult) string {
	return fmt.Sprintf("Successfully wrote to %s", r.Path)
}

// formatWriteFileJSON renders a WriteResult as JSON.
func formatWriteFileJSON(r *WriteResult) string {
	b, _ := json.Marshal(r)
	return string(b)
}

// formatCreateDirectoryText renders a CreateDirectoryResult in text format.
func formatCreateDirectoryText(r *CreateDirectoryResult) string {
	return fmt.Sprintf("Successfully created directory %s", r.Path)
}

// formatCreateDirectoryJSON renders a CreateDirectoryResult as JSON.
func formatCreateDirectoryJSON(r *CreateDirectoryResult) string {
	b, _ := json.Marshal(r)
	return string(b)
}

// formatMoveFileText renders a MoveResult in text format.
func formatMoveFileText(r *MoveResult) string {
	return fmt.Sprintf("Successfully moved %s to %s", r.Source, r.Destination)
}

// formatMoveFileJSON renders a MoveResult as JSON.
func formatMoveFileJSON(r *MoveResult) string {
	b, _ := json.Marshal(r)
	return string(b)
}

// formatAllowedDirectoriesText renders an AllowedDirectoriesResult in JSON format (already JSON-native).
func formatAllowedDirectoriesText(r *AllowedDirectoriesResult) string {
	// Current implementation returns JSON even in text mode for backward compat
	result := map[string]any{
		"allowed_directories": r.AllowedDirectories,
	}
	b, _ := json.MarshalIndent(result, "", "  ")
	return string(b)
}

// formatAllowedDirectoriesJSON renders an AllowedDirectoriesResult as JSON.
func formatAllowedDirectoriesJSON(r *AllowedDirectoriesResult) string {
	b, _ := json.Marshal(r)
	return string(b)
}

// formatGrepText renders a GrepResult in ripgrep-style text output.
// Format: path:linenum:text for matches, path-linenum-text for context lines.
// Groups separated by "--" when there are gaps between context windows.
func formatGrepText(r *GrepResult) string {
	if len(r.Matches) == 0 {
		return "No matches found"
	}

	var sb strings.Builder

	// Track the last line number emitted per file to know when to write a "--" separator.
	type fileState struct {
		lastLine int
	}
	fileStates := make(map[string]*fileState)

	for _, match := range r.Matches {
		st, exists := fileStates[match.Path]
		if !exists {
			st = &fileState{}
			fileStates[match.Path] = st
		}

		// Write separator if there's a gap since the last emitted line for this file.
		if st.lastLine > 0 {
			expectedNext := st.lastLine + 1
			firstLineOfWindow := match.LineNumber
			if len(match.ContextBefore) > 0 {
				firstLineOfWindow = match.ContextBefore[0].LineNumber
			}
			if firstLineOfWindow > expectedNext {
				sb.WriteString("--\n")
			}
		}

		// Context before
		for _, cl := range match.ContextBefore {
			sb.WriteString(fmt.Sprintf("%s-%d-%s\n", match.Path, cl.LineNumber, cl.LineText))
			st.lastLine = cl.LineNumber
		}

		// Match line
		sb.WriteString(fmt.Sprintf("%s:%d:%s\n", match.Path, match.LineNumber, match.LineText))
		st.lastLine = match.LineNumber

		// Context after
		for _, cl := range match.ContextAfter {
			sb.WriteString(fmt.Sprintf("%s-%d-%s\n", match.Path, cl.LineNumber, cl.LineText))
			st.lastLine = cl.LineNumber
		}
	}

	// Append summary line
	sb.WriteString(fmt.Sprintf("\n%d match(es) in %d file(s) searched %d file(s)",
		r.Summary.TotalMatches, r.Summary.FilesMatched, r.Summary.FilesSearched))
	if r.Summary.Truncated {
		sb.WriteString(" [truncated]")
	}

	return sb.String()
}

// formatGrepJSON renders a GrepResult as JSON.
func formatGrepJSON(r *GrepResult) string {
	b, _ := json.Marshal(r)
	return string(b)
}

// formatCopyFileText renders a CopyResult in text format.
func formatCopyFileText(r *CopyResult) string {
	return fmt.Sprintf("Successfully copied %d file(s) (%d bytes) to %s",
		r.Summary.FilesCopied, r.Summary.BytesCopied, r.Destination)
}

// formatCopyFileJSON renders a CopyResult as JSON.
func formatCopyFileJSON(r *CopyResult) string {
	b, _ := json.Marshal(r)
	return string(b)
}

// formatAppendFileText renders an AppendResult in text format.
func formatAppendFileText(r *AppendResult) string {
	if r.Created {
		return fmt.Sprintf("Successfully created and wrote %d bytes to %s", r.BytesWritten, r.Path)
	}
	return fmt.Sprintf("Successfully appended %d bytes to %s", r.BytesWritten, r.Path)
}

// formatAppendFileJSON renders an AppendResult as JSON.
func formatAppendFileJSON(r *AppendResult) string {
	b, _ := json.Marshal(r)
	return string(b)
}

// formatSymlinkText renders a SymlinkResult in text format.
func formatSymlinkText(r *SymlinkResult) string {
	return fmt.Sprintf("Successfully created symlink %s -> %s", r.Path, r.Target)
}

// formatSymlinkJSON renders a SymlinkResult as JSON.
func formatSymlinkJSON(r *SymlinkResult) string {
	b, _ := json.Marshal(r)
	return string(b)
}

// formatErrorJSON renders an error as structured JSON.
func formatErrorJSON(err error) string {
	resp := ErrorResponse{
		Error: ErrorDetail{
			Code:    "error",
			Message: err.Error(),
		},
	}
	b, _ := json.Marshal(resp)
	return string(b)
}
