package filesystem

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

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
