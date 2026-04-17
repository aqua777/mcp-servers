package filesystem

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatListDirectoryText(t *testing.T) {
	result := &ListDirectoryResult{
		Path: "/test/path",
		Entries: []DirectoryEntry{
			{Name: "file.txt", Type: "file", Size: nil},
			{Name: "subdir", Type: "directory", Size: nil},
		},
		Summary: DirectorySummary{
			Files:       1,
			Directories: 1,
		},
	}

	text := formatListDirectoryText(result)
	assert.Contains(t, text, "[FILE] file.txt")
	assert.Contains(t, text, "[DIR] subdir")
}

func TestFormatListDirectoryTextWithSizes(t *testing.T) {
	size := int64(100)
	result := &ListDirectoryResult{
		Path: "/test/path",
		Entries: []DirectoryEntry{
			{Name: "file.txt", Type: "file", Size: &size},
		},
		Summary: DirectorySummary{
			Files:       1,
			Directories: 0,
			TotalSize:   100,
		},
		SortBy: "name",
	}

	text := formatListDirectoryText(result)
	assert.Contains(t, text, "[FILE] file.txt (100 bytes)")
	assert.Contains(t, text, "Summary: 1 files, 0 directories, 100 bytes total")
}

func TestFormatListDirectoryJSON(t *testing.T) {
	size := int64(100)
	result := &ListDirectoryResult{
		Path: "/test/path",
		Entries: []DirectoryEntry{
			{Name: "file.txt", Type: "file", Size: &size},
		},
		Summary: DirectorySummary{
			Files:       1,
			Directories: 0,
			TotalSize:   100,
		},
		SortBy: "name",
	}

	jsonStr := formatListDirectoryJSON(result)

	var parsed ListDirectoryResult
	err := json.Unmarshal([]byte(jsonStr), &parsed)
	require.NoError(t, err)
	assert.Equal(t, "/test/path", parsed.Path)
	assert.Equal(t, 1, len(parsed.Entries))
	assert.Equal(t, "file.txt", parsed.Entries[0].Name)
}

func TestFormatSearchFilesText(t *testing.T) {
	result := &SearchResult{
		Root:    "/test",
		Pattern: "*.go",
		Matches: []SearchMatch{
			{Path: "/test/main.go", Type: "file"},
			{Path: "/test/util.go", Type: "file"},
		},
		Summary: SearchSummary{
			TotalMatches: 2,
		},
	}

	text := formatSearchFilesText(result)
	assert.Contains(t, text, "/test/main.go")
	assert.Contains(t, text, "/test/util.go")
}

func TestFormatSearchFilesJSON(t *testing.T) {
	result := &SearchResult{
		Root:    "/test",
		Pattern: "*.go",
		Matches: []SearchMatch{
			{Path: "/test/main.go", Type: "file"},
		},
		Summary: SearchSummary{
			TotalMatches: 1,
		},
	}

	jsonStr := formatSearchFilesJSON(result)

	var parsed SearchResult
	err := json.Unmarshal([]byte(jsonStr), &parsed)
	require.NoError(t, err)
	assert.Equal(t, "/test", parsed.Root)
	assert.Equal(t, 1, parsed.Summary.TotalMatches)
}

func TestFormatDirectoryTreeJSON(t *testing.T) {
	result := &DirectoryTreeResult{
		Root: &Node{
			Name: "root",
			Type: "directory",
			Children: []*Node{
				{Name: "file.txt", Type: "file"},
			},
		},
		Summary: DirectorySummary{
			Files:       1,
			Directories: 1,
		},
	}

	jsonStr := formatDirectoryTreeJSON(result)
	assert.Contains(t, jsonStr, "root")
	assert.Contains(t, jsonStr, "file.txt")
}

func TestFormatReadFileText(t *testing.T) {
	result := &ReadFileResult{
		Path:    "/test/file.txt",
		Size:    100,
		Lines:   5,
		Content: "line1\nline2",
	}

	text := formatReadFileText(result)
	assert.Equal(t, "line1\nline2", text)
}

func TestFormatReadFileJSON(t *testing.T) {
	result := &ReadFileResult{
		Path:    "/test/file.txt",
		Size:    100,
		Lines:   5,
		Content: "line1\nline2",
	}

	jsonStr := formatReadFileJSON(result)

	var parsed ReadFileResult
	err := json.Unmarshal([]byte(jsonStr), &parsed)
	require.NoError(t, err)
	assert.Equal(t, "/test/file.txt", parsed.Path)
	assert.Equal(t, int64(100), parsed.Size)
	assert.Equal(t, 5, parsed.Lines)
}

func TestFormatReadMultipleFilesText(t *testing.T) {
	result := &ReadMultipleFilesResult{
		Files: []FileReadResult{
			{Path: "/test/file1.txt", Status: "ok", Content: "content1"},
			{Path: "/test/file2.txt", Status: "error", Error: "not found"},
		},
		Summary: ReadMultipleFilesSummary{
			Succeeded: 1,
			Failed:    1,
			Total:     2,
		},
	}

	text := formatReadMultipleFilesText(result)
	assert.Contains(t, text, "--- /test/file1.txt ---")
	assert.Contains(t, text, "content1")
	assert.Contains(t, text, "/test/file2.txt: Error:")
}

func TestFormatReadMultipleFilesJSON(t *testing.T) {
	result := &ReadMultipleFilesResult{
		Files: []FileReadResult{
			{Path: "/test/file1.txt", Status: "ok", Content: "content1"},
		},
		Summary: ReadMultipleFilesSummary{
			Succeeded: 1,
			Failed:    0,
			Total:     1,
		},
	}

	jsonStr := formatReadMultipleFilesJSON(result)

	var parsed ReadMultipleFilesResult
	err := json.Unmarshal([]byte(jsonStr), &parsed)
	require.NoError(t, err)
	assert.Equal(t, 1, len(parsed.Files))
	assert.Equal(t, "ok", parsed.Files[0].Status)
}

func TestFormatEditFileText(t *testing.T) {
	result := &EditFileResult{
		Path:           "/test/file.txt",
		Status:         "applied",
		EditsApplied:   2,
		EditsRequested: 2,
	}

	text := formatEditFileText(result)
	assert.Contains(t, text, "Successfully edited")
	assert.Contains(t, text, "/test/file.txt")
}

func TestFormatEditFileJSON(t *testing.T) {
	result := &EditFileResult{
		Path:           "/test/file.txt",
		Status:         "dry_run",
		Diff:           "--- a\n+++ b",
		EditsApplied:   2,
		EditsRequested: 2,
	}

	jsonStr := formatEditFileJSON(result)

	var parsed EditFileResult
	err := json.Unmarshal([]byte(jsonStr), &parsed)
	require.NoError(t, err)
	assert.Equal(t, "dry_run", parsed.Status)
	assert.Equal(t, 2, parsed.EditsApplied)
}

func TestFormatFileInfoJSON(t *testing.T) {
	result := &FileInfoResult{
		Path:         "/test/file.txt",
		Name:         "file.txt",
		Size:         1024,
		Type:         "file",
		Permissions:  "-rw-r--r--",
		ModifiedTime: "2024-01-01T00:00:00Z",
		IsSymlink:    false,
	}

	jsonStr := formatFileInfoJSON(result)

	var parsed FileInfoResult
	err := json.Unmarshal([]byte(jsonStr), &parsed)
	require.NoError(t, err)
	assert.Equal(t, "/test/file.txt", parsed.Path)
	assert.Equal(t, int64(1024), parsed.Size)
}

func TestFormatWriteFileText(t *testing.T) {
	result := &WriteResult{
		Path:         "/test/file.txt",
		Status:       "ok",
		BytesWritten: 100,
	}

	text := formatWriteFileText(result)
	assert.Contains(t, text, "Successfully wrote")
	assert.Contains(t, text, "/test/file.txt")
}

func TestFormatWriteFileJSON(t *testing.T) {
	result := &WriteResult{
		Path:         "/test/file.txt",
		Status:       "ok",
		BytesWritten: 100,
	}

	jsonStr := formatWriteFileJSON(result)

	var parsed WriteResult
	err := json.Unmarshal([]byte(jsonStr), &parsed)
	require.NoError(t, err)
	assert.Equal(t, 100, parsed.BytesWritten)
}

func TestFormatCreateDirectoryJSON(t *testing.T) {
	result := &CreateDirectoryResult{
		Path:    "/test/newdir",
		Status:  "ok",
		Created: true,
	}

	jsonStr := formatCreateDirectoryJSON(result)

	var parsed CreateDirectoryResult
	err := json.Unmarshal([]byte(jsonStr), &parsed)
	require.NoError(t, err)
	assert.True(t, parsed.Created)
}

func TestFormatMoveFileJSON(t *testing.T) {
	result := &MoveResult{
		Source:      "/test/old.txt",
		Destination: "/test/new.txt",
		Status:      "ok",
	}

	jsonStr := formatMoveFileJSON(result)

	var parsed MoveResult
	err := json.Unmarshal([]byte(jsonStr), &parsed)
	require.NoError(t, err)
	assert.Equal(t, "/test/old.txt", parsed.Source)
	assert.Equal(t, "/test/new.txt", parsed.Destination)
}

func TestFormatCopyFileText(t *testing.T) {
	result := &CopyResult{
		Source:      "/src",
		Destination: "/dst",
		Status:      "ok",
		Entries:     []CopiedEntry{{Source: "/src/a.go", Destination: "/dst/a.go", Type: "file"}},
		Summary:     CopyFileSummary{FilesCopied: 1, BytesCopied: 42},
	}

	text := formatCopyFileText(result)
	assert.Contains(t, text, "Successfully copied")
	assert.Contains(t, text, "1 file(s)")
	assert.Contains(t, text, "42 bytes")
	assert.Contains(t, text, "/dst")
}

func TestFormatCopyFileJSON(t *testing.T) {
	result := &CopyResult{
		Source:      "/src",
		Destination: "/dst",
		Status:      "ok",
		Entries:     []CopiedEntry{{Source: "/src/a.go", Destination: "/dst/a.go", Type: "file"}},
		Summary:     CopyFileSummary{FilesCopied: 1, BytesCopied: 42},
	}

	jsonStr := formatCopyFileJSON(result)

	var parsed CopyResult
	err := json.Unmarshal([]byte(jsonStr), &parsed)
	require.NoError(t, err)
	assert.Equal(t, "ok", parsed.Status)
	assert.Equal(t, 1, parsed.Summary.FilesCopied)
	assert.EqualValues(t, 42, parsed.Summary.BytesCopied)
}

func TestFormatAppendFileTextExisting(t *testing.T) {
	result := &AppendResult{
		Path:         "/test/file.txt",
		Status:       "ok",
		BytesWritten: 20,
		Created:      false,
	}

	text := formatAppendFileText(result)
	assert.Contains(t, text, "appended")
	assert.Contains(t, text, "20 bytes")
	assert.Contains(t, text, "/test/file.txt")
}

func TestFormatAppendFileTextCreated(t *testing.T) {
	result := &AppendResult{
		Path:         "/test/new.txt",
		Status:       "ok",
		BytesWritten: 10,
		Created:      true,
	}

	text := formatAppendFileText(result)
	assert.Contains(t, text, "created")
	assert.Contains(t, text, "10 bytes")
}

func TestFormatAppendFileJSON(t *testing.T) {
	result := &AppendResult{
		Path:         "/test/file.txt",
		Status:       "ok",
		BytesWritten: 20,
		Created:      false,
	}

	jsonStr := formatAppendFileJSON(result)

	var parsed AppendResult
	err := json.Unmarshal([]byte(jsonStr), &parsed)
	require.NoError(t, err)
	assert.Equal(t, "/test/file.txt", parsed.Path)
	assert.Equal(t, 20, parsed.BytesWritten)
	assert.False(t, parsed.Created)
}

func TestFormatSymlinkText(t *testing.T) {
	result := &SymlinkResult{
		Path:   "/test/link.txt",
		Target: "/test/real.txt",
		Status: "ok",
	}

	text := formatSymlinkText(result)
	assert.Contains(t, text, "Successfully created symlink")
	assert.Contains(t, text, "/test/link.txt")
	assert.Contains(t, text, "/test/real.txt")
}

func TestFormatSymlinkJSON(t *testing.T) {
	result := &SymlinkResult{
		Path:   "/test/link.txt",
		Target: "/test/real.txt",
		Status: "ok",
	}

	jsonStr := formatSymlinkJSON(result)

	var parsed SymlinkResult
	err := json.Unmarshal([]byte(jsonStr), &parsed)
	require.NoError(t, err)
	assert.Equal(t, "/test/link.txt", parsed.Path)
	assert.Equal(t, "/test/real.txt", parsed.Target)
	assert.Equal(t, "ok", parsed.Status)
}

func TestFormatAllowedDirectoriesJSON(t *testing.T) {
	result := &AllowedDirectoriesResult{
		AllowedDirectories: []string{"/test", "/home"},
	}

	jsonStr := formatAllowedDirectoriesJSON(result)

	var parsed AllowedDirectoriesResult
	err := json.Unmarshal([]byte(jsonStr), &parsed)
	require.NoError(t, err)
	assert.Equal(t, 2, len(parsed.AllowedDirectories))
}
