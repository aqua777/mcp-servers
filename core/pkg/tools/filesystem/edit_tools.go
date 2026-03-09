package filesystem

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/sergi/go-diff/diffmatchpatch"
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
			},
			"required": []string{"path"},
		},
	}, s.handleGetFileInfo)

	s.server.AddTool(&mcp.Tool{
		Name:        "list_allowed_directories",
		Description: `List all directories the server is allowed to access.`,
		InputSchema: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
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
	}
	if err := json.Unmarshal(request.Params.Arguments, &args); err != nil {
		return errorResult(fmt.Sprintf("Invalid arguments: %v", err)), nil
	}

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

	for i, edit := range args.Edits {
		if !strings.Contains(content, edit.OldText) {
			return errorResult(fmt.Sprintf("Edit %d failed: oldText not found in file", i+1)), nil
		}
		content = strings.Replace(content, edit.OldText, edit.NewText, 1) // Replace first occurrence only (similar to TS implementation if not global)
		// Note: The TS implementation uses advanced parsing/matching. We are using simple string replacement here as a baseline.
	}

	if args.DryRun {
		dmp := diffmatchpatch.New()
		diffs := dmp.DiffMain(originalContent, content, false)
		diffText := dmp.DiffPrettyText(diffs)

		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{
				Text: fmt.Sprintf("Dry run diff:\n%s", diffText),
			}},
		}, nil
	}

	err = os.WriteFile(validPath, []byte(content), 0644)
	if err != nil {
		return errorResult(fmt.Sprintf("Error writing file: %v", err)), nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{
			Text: fmt.Sprintf("Successfully edited %s", args.Path),
		}},
	}, nil
}

func (s *FilesystemServer) handleGetFileInfo(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(request.Params.Arguments, &args); err != nil {
		return errorResult(fmt.Sprintf("Invalid arguments: %v", err)), nil
	}

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

	// Getting advanced timestamps (creation, access) is OS-specific in Go.
	// We'll provide standard info.
	result := map[string]any{
		"size":         info.Size(),
		"modifiedTime": info.ModTime(),
		"type":         entryType,
		"permissions":  info.Mode().String(),
	}

	jsonBytes, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return errorResult(fmt.Sprintf("JSON marshal error: %v", err)), nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{
			Text: string(jsonBytes),
		}},
	}, nil
}

func (s *FilesystemServer) handleListAllowedDirectories(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dirs := s.GetAllowedDirectories()

	result := map[string]any{
		"allowedDirectories": dirs,
	}

	jsonBytes, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return errorResult(fmt.Sprintf("JSON marshal error: %v", err)), nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{
			Text: string(jsonBytes),
		}},
	}, nil
}
