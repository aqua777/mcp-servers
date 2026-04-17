package filesystem

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func (s *FilesystemServer) registerWriteTools() {
	s.server.AddTool(&mcp.Tool{
		Name:        "create_directory",
		Description: `Create a new directory or ensure it exists. Creates parent directories if needed. Succeeds silently if directory exists.`,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "Path to the directory to create",
				},
				"format": map[string]any{
					"type":        "string",
					"enum":        []string{"text", "json"},
					"description": "Output format (default: server setting)",
				},
			},
			"required": []string{"path"},
		},
	}, s.handleCreateDirectory)

	s.server.AddTool(&mcp.Tool{
		Name:        "write_file",
		Description: `Create a new file or overwrite an existing one.`,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "Path to the file to write",
				},
				"content": map[string]any{
					"type":        "string",
					"description": "Content to write to the file",
				},
				"format": map[string]any{
					"type":        "string",
					"enum":        []string{"text", "json"},
					"description": "Output format (default: server setting)",
				},
			},
			"required": []string{"path", "content"},
		},
	}, s.handleWriteFile)

	s.server.AddTool(&mcp.Tool{
		Name:        "move_file",
		Description: `Move or rename files and directories. Fails if destination already exists.`,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"source": map[string]any{
					"type":        "string",
					"description": "Source path",
				},
				"destination": map[string]any{
					"type":        "string",
					"description": "Destination path",
				},
				"format": map[string]any{
					"type":        "string",
					"enum":        []string{"text", "json"},
					"description": "Output format (default: server setting)",
				},
			},
			"required": []string{"source", "destination"},
		},
	}, s.handleMoveFile)
}

func (s *FilesystemServer) handleCreateDirectory(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args struct {
		Path   string `json:"path"`
		Format string `json:"format"`
	}
	if err := json.Unmarshal(request.Params.Arguments, &args); err != nil {
		return errorResult(fmt.Sprintf("Invalid arguments: %v", err)), nil
	}

	format := s.resolveFormat(args.Format)
	validPath, err := s.validatePath(args.Path)
	if err != nil {
		return errorResult(err.Error()), nil
	}

	// Check if directory already exists
	created := true
	if _, err := os.Stat(validPath); err == nil {
		created = false
	}

	err = os.MkdirAll(validPath, 0755)
	if err != nil {
		return errorResult(fmt.Sprintf("Error creating directory: %v", err)), nil
	}

	result := &CreateDirectoryResult{
		Path:    validPath,
		Status:  "ok",
		Created: created,
	}

	var text string
	if format == FormatJSON {
		text = formatCreateDirectoryJSON(result)
	} else {
		text = formatCreateDirectoryText(result)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{
			Text: text,
		}},
	}, nil
}

func (s *FilesystemServer) handleWriteFile(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args struct {
		Path    string `json:"path"`
		Content string `json:"content"`
		Format  string `json:"format"`
	}
	if err := json.Unmarshal(request.Params.Arguments, &args); err != nil {
		return errorResult(fmt.Sprintf("Invalid arguments: %v", err)), nil
	}

	format := s.resolveFormat(args.Format)
	validPath, err := s.validatePath(args.Path)
	if err != nil {
		return errorResult(err.Error()), nil
	}

	err = os.WriteFile(validPath, []byte(args.Content), 0644)
	if err != nil {
		return errorResult(fmt.Sprintf("Error writing file: %v", err)), nil
	}

	result := &WriteResult{
		Path:         validPath,
		Status:       "ok",
		BytesWritten: len(args.Content),
	}

	var text string
	if format == FormatJSON {
		text = formatWriteFileJSON(result)
	} else {
		text = formatWriteFileText(result)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{
			Text: text,
		}},
	}, nil
}

func (s *FilesystemServer) registerCopyAppendSymlinkTools() {
	s.server.AddTool(&mcp.Tool{
		Name: ToolCopyFile,
		Description: `Copy files or directories. Source supports glob patterns (e.g. /dir/*.go).
For glob sources or directory sources, all matches are copied into the destination directory.
Requires 'recursive' flag to copy directories.`,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"source": map[string]any{
					"type":        "string",
					"description": "Source path or glob pattern (e.g. /dir/*.go)",
				},
				"destination": map[string]any{
					"type":        "string",
					"description": "Destination file path (single file copy) or destination directory (glob/recursive copy)",
				},
				"recursive": map[string]any{
					"type":        "boolean",
					"description": "Allow recursive directory copy (default: false). Required when source is or matches a directory.",
					"default":     false,
				},
				"excludePatterns": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "string",
					},
					"description": "Glob patterns to exclude during recursive copy",
				},
				"format": map[string]any{
					"type":        "string",
					"enum":        []string{"text", "json"},
					"description": "Output format (default: server setting)",
				},
			},
			"required": []string{"source", "destination"},
		},
	}, s.handleCopyFile)

	s.server.AddTool(&mcp.Tool{
		Name:        ToolAppendFile,
		Description: `Append content to an existing file, or create it if it does not exist.`,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "File to append to",
				},
				"content": map[string]any{
					"type":        "string",
					"description": "Content to append",
				},
				"format": map[string]any{
					"type":        "string",
					"enum":        []string{"text", "json"},
					"description": "Output format (default: server setting)",
				},
			},
			"required": []string{"path", "content"},
		},
	}, s.handleAppendFile)

	s.server.AddTool(&mcp.Tool{
		Name:        ToolCreateSymlink,
		Description: `Create a symbolic link. Both the symlink path and its target must be within allowed directories.`,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"target": map[string]any{
					"type":        "string",
					"description": "Path the symlink points to",
				},
				"path": map[string]any{
					"type":        "string",
					"description": "Path of the symlink to create",
				},
				"format": map[string]any{
					"type":        "string",
					"enum":        []string{"text", "json"},
					"description": "Output format (default: server setting)",
				},
			},
			"required": []string{"target", "path"},
		},
	}, s.handleCreateSymlink)
}

// isGlobPattern reports whether s contains any doublestar glob metacharacters.
func isGlobPattern(s string) bool {
	return strings.ContainsAny(s, "*?[")
}

// copyFileContent copies a single regular file from src to dst, creating parent dirs as needed.
// Returns the number of bytes copied.
func copyFileContent(src, dst string) (int64, error) {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return 0, fmt.Errorf("error creating parent directory for %s: %w", dst, err)
	}

	in, err := os.Open(src)
	if err != nil {
		return 0, fmt.Errorf("error opening source file: %w", err)
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return 0, fmt.Errorf("error creating destination file: %w", err)
	}
	defer out.Close()

	n, err := io.Copy(out, in)
	if err != nil {
		return n, fmt.Errorf("error copying file: %w", err)
	}
	return n, nil
}

// copyDir recursively copies a directory from src to dst, respecting excludePatterns.
// It appends entries to the provided slices and increments counters.
func copyDir(src, dst string, excludePatterns []string, entries *[]CopiedEntry, summary *CopyFileSummary) error {
	fileSys := os.DirFS(src)
	return fs.WalkDir(fileSys, ".", func(relPath string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		// Apply exclude patterns
		for _, excl := range excludePatterns {
			matched, _ := doublestar.Match(excl, relPath)
			if matched {
				if d.IsDir() {
					return fs.SkipDir
				}
				return nil
			}
		}

		srcPath := filepath.Join(src, filepath.FromSlash(relPath))
		dstPath := filepath.Join(dst, filepath.FromSlash(relPath))

		if d.IsDir() {
			if err := os.MkdirAll(dstPath, 0755); err != nil {
				return fmt.Errorf("error creating directory %s: %w", dstPath, err)
			}
			if relPath != "." {
				*entries = append(*entries, CopiedEntry{Source: srcPath, Destination: dstPath, Type: "directory"})
				summary.DirectoriesMade++
			}
			return nil
		}

		n, err := copyFileContent(srcPath, dstPath)
		if err != nil {
			return err
		}
		*entries = append(*entries, CopiedEntry{Source: srcPath, Destination: dstPath, Type: "file"})
		summary.FilesCopied++
		summary.BytesCopied += n
		return nil
	})
}

func (s *FilesystemServer) handleCopyFile(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args struct {
		Source          string   `json:"source"`
		Destination     string   `json:"destination"`
		Recursive       bool     `json:"recursive"`
		ExcludePatterns []string `json:"excludePatterns"`
		Format          string   `json:"format"`
	}
	if err := json.Unmarshal(request.Params.Arguments, &args); err != nil {
		return errorResult(fmt.Sprintf("Invalid arguments: %v", err)), nil
	}

	format := s.resolveFormat(args.Format)

	validDest, err := s.validatePath(args.Destination)
	if err != nil {
		return errorResult(fmt.Sprintf("Destination validation error: %v", err)), nil
	}

	var summary CopyFileSummary
	var entries []CopiedEntry

	if isGlobPattern(args.Source) {
		// Glob mode: source is a glob pattern, destination must be a directory
		matches, err := doublestar.FilepathGlob(args.Source)
		if err != nil {
			return errorResult(fmt.Sprintf("Invalid glob pattern: %v", err)), nil
		}
		if len(matches) == 0 {
			return errorResult(fmt.Sprintf("No files match pattern: %s", args.Source)), nil
		}

		// Validate all matches are within allowed directories
		for _, match := range matches {
			if _, err := s.validatePath(match); err != nil {
				return errorResult(fmt.Sprintf("Source path validation error for %s: %v", match, err)), nil
			}
		}

		// Ensure destination directory exists
		if err := os.MkdirAll(validDest, 0755); err != nil {
			return errorResult(fmt.Sprintf("Error creating destination directory: %v", err)), nil
		}

		for _, match := range matches {
			info, err := os.Stat(match)
			if err != nil {
				return errorResult(fmt.Sprintf("Error accessing %s: %v", match, err)), nil
			}

			dstPath := filepath.Join(validDest, filepath.Base(match))

			if info.IsDir() {
				if !args.Recursive {
					return errorResult(fmt.Sprintf("Source %s is a directory: use recursive=true to copy directories", match)), nil
				}
				if err := copyDir(match, dstPath, args.ExcludePatterns, &entries, &summary); err != nil {
					return errorResult(fmt.Sprintf("Error copying directory %s: %v", match, err)), nil
				}
			} else {
				n, err := copyFileContent(match, dstPath)
				if err != nil {
					return errorResult(err.Error()), nil
				}
				entries = append(entries, CopiedEntry{Source: match, Destination: dstPath, Type: "file"})
				summary.FilesCopied++
				summary.BytesCopied += n
			}
		}
	} else {
		// Plain path mode
		validSrc, err := s.validatePath(args.Source)
		if err != nil {
			return errorResult(fmt.Sprintf("Source validation error: %v", err)), nil
		}

		info, err := os.Stat(validSrc)
		if err != nil {
			return errorResult(fmt.Sprintf("Error accessing source: %v", err)), nil
		}

		if info.IsDir() {
			if !args.Recursive {
				return errorResult(fmt.Sprintf("Source %s is a directory: use recursive=true to copy directories", args.Source)), nil
			}
			dstPath := validDest
			if err := copyDir(validSrc, dstPath, args.ExcludePatterns, &entries, &summary); err != nil {
				return errorResult(fmt.Sprintf("Error copying directory: %v", err)), nil
			}
		} else {
			// Single file: destination must not exist
			if _, err := os.Stat(validDest); err == nil {
				return errorResult(fmt.Sprintf("Destination %s already exists", args.Destination)), nil
			}
			n, err := copyFileContent(validSrc, validDest)
			if err != nil {
				return errorResult(err.Error()), nil
			}
			entries = append(entries, CopiedEntry{Source: validSrc, Destination: validDest, Type: "file"})
			summary.FilesCopied++
			summary.BytesCopied += n
		}
	}

	result := &CopyResult{
		Source:      args.Source,
		Destination: validDest,
		Status:      "ok",
		Entries:     entries,
		Summary:     summary,
	}

	var text string
	if format == FormatJSON {
		text = formatCopyFileJSON(result)
	} else {
		text = formatCopyFileText(result)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: text}},
	}, nil
}

func (s *FilesystemServer) handleAppendFile(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args struct {
		Path    string `json:"path"`
		Content string `json:"content"`
		Format  string `json:"format"`
	}
	if err := json.Unmarshal(request.Params.Arguments, &args); err != nil {
		return errorResult(fmt.Sprintf("Invalid arguments: %v", err)), nil
	}

	format := s.resolveFormat(args.Format)
	validPath, err := s.validatePath(args.Path)
	if err != nil {
		return errorResult(err.Error()), nil
	}

	_, statErr := os.Stat(validPath)
	created := os.IsNotExist(statErr)

	f, err := os.OpenFile(validPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return errorResult(fmt.Sprintf("Error opening file: %v", err)), nil
	}
	defer f.Close()

	n, err := f.WriteString(args.Content)
	if err != nil {
		return errorResult(fmt.Sprintf("Error writing to file: %v", err)), nil
	}

	result := &AppendResult{
		Path:         validPath,
		Status:       "ok",
		BytesWritten: n,
		Created:      created,
	}

	var text string
	if format == FormatJSON {
		text = formatAppendFileJSON(result)
	} else {
		text = formatAppendFileText(result)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: text}},
	}, nil
}

func (s *FilesystemServer) handleCreateSymlink(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args struct {
		Target string `json:"target"`
		Path   string `json:"path"`
		Format string `json:"format"`
	}
	if err := json.Unmarshal(request.Params.Arguments, &args); err != nil {
		return errorResult(fmt.Sprintf("Invalid arguments: %v", err)), nil
	}

	format := s.resolveFormat(args.Format)

	validPath, err := s.validatePath(args.Path)
	if err != nil {
		return errorResult(fmt.Sprintf("Symlink path validation error: %v", err)), nil
	}

	validTarget, err := s.validatePath(args.Target)
	if err != nil {
		return errorResult(fmt.Sprintf("Symlink target validation error: %v", err)), nil
	}

	if _, err := os.Lstat(validPath); err == nil {
		return errorResult(fmt.Sprintf("Path %s already exists", args.Path)), nil
	}

	if err := os.MkdirAll(filepath.Dir(validPath), 0755); err != nil {
		return errorResult(fmt.Sprintf("Error creating parent directory: %v", err)), nil
	}

	if err := os.Symlink(validTarget, validPath); err != nil {
		return errorResult(fmt.Sprintf("Error creating symlink: %v", err)), nil
	}

	result := &SymlinkResult{
		Path:   validPath,
		Target: validTarget,
		Status: "ok",
	}

	var text string
	if format == FormatJSON {
		text = formatSymlinkJSON(result)
	} else {
		text = formatSymlinkText(result)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: text}},
	}, nil
}

func (s *FilesystemServer) handleMoveFile(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args struct {
		Source      string `json:"source"`
		Destination string `json:"destination"`
		Format      string `json:"format"`
	}
	if err := json.Unmarshal(request.Params.Arguments, &args); err != nil {
		return errorResult(fmt.Sprintf("Invalid arguments: %v", err)), nil
	}

	format := s.resolveFormat(args.Format)
	validSource, err := s.validatePath(args.Source)
	if err != nil {
		return errorResult(fmt.Sprintf("Source validation error: %v", err)), nil
	}

	validDest, err := s.validatePath(args.Destination)
	if err != nil {
		return errorResult(fmt.Sprintf("Destination validation error: %v", err)), nil
	}

	if _, err := os.Stat(validDest); err == nil {
		return errorResult(fmt.Sprintf("Destination %s already exists", args.Destination)), nil
	}

	// Ensure parent directory of destination exists
	err = os.MkdirAll(filepath.Dir(validDest), 0755)
	if err != nil {
		return errorResult(fmt.Sprintf("Error creating destination parent directory: %v", err)), nil
	}

	err = os.Rename(validSource, validDest)
	if err != nil {
		return errorResult(fmt.Sprintf("Error moving file: %v", err)), nil
	}

	result := &MoveResult{
		Source:      validSource,
		Destination: validDest,
		Status:      "ok",
	}

	var text string
	if format == FormatJSON {
		text = formatMoveFileJSON(result)
	} else {
		text = formatMoveFileText(result)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{
			Text: text,
		}},
	}, nil
}
