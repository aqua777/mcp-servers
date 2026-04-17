package filesystem

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/pmezard/go-difflib/difflib"
)

func (s *FilesystemServer) registerEditTools() {
	s.server.AddTool(&mcp.Tool{
		Name:        "edit_file",
		Description: `Make selective edits using advanced pattern matching and formatting. Returns detailed diff for dry runs, otherwise applies changes.`,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "File to edit",
				},
				"edits": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"oldText": map[string]any{
								"type": "string",
							},
							"newText": map[string]any{
								"type": "string",
							},
						},
						"required": []string{"oldText", "newText"},
					},
					"description": "List of edit operations",
				},
				"dryRun": map[string]any{
					"type":        "boolean",
					"description": "Preview changes without applying (default: false)",
					"default":     false,
				},
				"format": map[string]any{
					"type":        "string",
					"enum":        []string{"text", "json"},
					"description": "Output format (default: server setting)",
				},
			},
			"required": []string{"path", "edits"},
		},
	}, s.handleEditFile)

	s.server.AddTool(&mcp.Tool{
		Name:        "get_file_info",
		Description: `Get detailed file/directory metadata.`,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "Path to the file or directory",
				},
				"format": map[string]any{
					"type":        "string",
					"enum":        []string{"text", "json"},
					"description": "Output format (default: server setting)",
				},
			},
			"required": []string{"path"},
		},
	}, s.handleGetFileInfo)

	s.server.AddTool(&mcp.Tool{
		Name:        "list_allowed_directories",
		Description: `List all directories the server is allowed to access.`,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"format": map[string]any{
					"type":        "string",
					"enum":        []string{"text", "json"},
					"description": "Output format (default: server setting)",
				},
			},
		},
	}, s.handleListAllowedDirectories)
}

func (s *FilesystemServer) handleEditFile(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	type Edit struct {
		OldText string `json:"oldText"`
		NewText string `json:"newText"`
	}
	var args struct {
		Path   string `json:"path"`
		Edits  []Edit `json:"edits"`
		DryRun bool   `json:"dryRun"`
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

	contentBytes, err := os.ReadFile(validPath)
	if err != nil {
		return errorResult(fmt.Sprintf("Error reading file: %v", err)), nil
	}

	content := string(contentBytes)
	originalContent := content
	editsApplied := 0

	for i, edit := range args.Edits {
		if !strings.Contains(content, edit.OldText) {
			return errorResult(fmt.Sprintf("Edit %d failed: oldText not found in file", i+1)), nil
		}
		content = strings.Replace(content, edit.OldText, edit.NewText, 1)
		editsApplied++
	}

	var diffText string
	if args.DryRun || format == FormatJSON {
		diff := difflib.UnifiedDiff{
			A:        difflib.SplitLines(originalContent),
			B:        difflib.SplitLines(content),
			FromFile: args.Path,
			ToFile:   args.Path,
			Context:  3,
		}
		dt, err := difflib.GetUnifiedDiffString(diff)
		if err != nil {
			return errorResult(fmt.Sprintf("Error generating diff: %v", err)), nil
		}
		diffText = dt
	}

	if args.DryRun {
		result := &EditFileResult{
			Path:           validPath,
			Status:         "dry_run",
			Diff:           diffText,
			EditsApplied:   editsApplied,
			EditsRequested: len(args.Edits),
		}

		var text string
		if format == FormatJSON {
			text = formatEditFileJSON(result)
		} else {
			text = formatEditFileText(result)
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{
				Text: text,
			}},
		}, nil
	}

	err = os.WriteFile(validPath, []byte(content), 0644)
	if err != nil {
		return errorResult(fmt.Sprintf("Error writing file: %v", err)), nil
	}

	result := &EditFileResult{
		Path:           validPath,
		Status:         "applied",
		EditsApplied:   editsApplied,
		EditsRequested: len(args.Edits),
	}

	var text string
	if format == FormatJSON {
		text = formatEditFileJSON(result)
	} else {
		text = formatEditFileText(result)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{
			Text: text,
		}},
	}, nil
}

func (s *FilesystemServer) handleGetFileInfo(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

	info, err := os.Stat(validPath)
	if err != nil {
		return errorResult(fmt.Sprintf("Error getting file info: %v", err)), nil
	}

	entryType := "file"
	if info.IsDir() {
		entryType = "directory"
	}

	result := &FileInfoResult{
		Path:         validPath,
		Name:         info.Name(),
		Size:         info.Size(),
		Type:         entryType,
		Permissions:  info.Mode().String(),
		ModifiedTime: info.ModTime().Format("2006-01-02T15:04:05Z07:00"),
		IsSymlink:    info.Mode()&os.ModeSymlink != 0,
	}

	var text string
	if format == FormatJSON {
		text = formatFileInfoJSON(result)
	} else {
		text = formatFileInfoText(result)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{
			Text: text,
		}},
	}, nil
}

func (s *FilesystemServer) handleListAllowedDirectories(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args struct {
		Format string `json:"format"`
	}
	if err := json.Unmarshal(request.Params.Arguments, &args); err != nil {
		return errorResult(fmt.Sprintf("Invalid arguments: %v", err)), nil
	}

	format := s.resolveFormat(args.Format)
	dirs := s.GetAllowedDirectories()

	result := &AllowedDirectoriesResult{
		AllowedDirectories: dirs,
	}

	var text string
	if format == FormatJSON {
		text = formatAllowedDirectoriesJSON(result)
	} else {
		text = formatAllowedDirectoriesText(result)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{
			Text: text,
		}},
	}, nil
}
