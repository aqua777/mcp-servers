package filesystem

import (
	"errors"
	"path/filepath"
	"strings"
)

// IsPathWithinAllowedDirectories checks if an absolute path is within any of the allowed directories.
// It normalizes both the input path and allowed directories before checking.
func IsPathWithinAllowedDirectories(absolutePath string, allowedDirectories []string) (bool, error) {
	if absolutePath == "" || len(allowedDirectories) == 0 {
		return false, nil
	}

	// Reject null bytes
	if strings.Contains(absolutePath, "\x00") {
		return false, nil
	}

	// Normalize the input path
	normalizedPath := filepath.Clean(absolutePath)
	if !filepath.IsAbs(normalizedPath) {
		return false, errors.New("path must be absolute after normalization")
	}

	for _, dir := range allowedDirectories {
		if dir == "" {
			continue
		}

		if strings.Contains(dir, "\x00") {
			continue
		}

		normalizedDir := filepath.Clean(dir)
		if !filepath.IsAbs(normalizedDir) {
			return false, errors.New("allowed directories must be absolute paths after normalization")
		}

		// Exact match
		if normalizedPath == normalizedDir {
			return true, nil
		}

		// Subdirectory match
		// Ensure it ends with a separator to prevent partial name matches (e.g., /foo/bar matching /foo/bar2)
		prefix := normalizedDir
		if !strings.HasSuffix(prefix, string(filepath.Separator)) {
			prefix += string(filepath.Separator)
		}

		// On Windows, paths are case-insensitive, but for now we rely on strict prefix matching 
		// or filepath.Rel to be safer.
		// A more robust way in Go is to use filepath.Rel
		rel, err := filepath.Rel(normalizedDir, normalizedPath)
		if err == nil && !strings.HasPrefix(rel, "..") && rel != ".." {
			return true, nil
		}
	}

	return false, nil
}
