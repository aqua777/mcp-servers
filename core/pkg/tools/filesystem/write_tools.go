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
		Name: "create_directory",
		Description: `Create a new directory or ensure it exists. Creates parent directories if needed. Succeeds silently if directory exists.`,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type": "string",
					"description": "Path to the directory to create",
				},
			},
			"required": []string{"path"},
		},
	}, func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

		err = os.MkdirAll(validPath, 0755)
		if err != nil {
			return errorResult(fmt.Sprintf("Error creating directory: %v", err)), nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{
				Text: fmt.Sprintf("Successfully created directory %s", args.Path),
			}},
		}, nil
	})

	s.server.AddTool(&mcp.Tool{
		Name: "write_file",
		Description: `Create a new file or overwrite an existing one.`,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type": "string",
					"description": "Path to the file to write",
				},
				"content": map[string]any{
					"type": "string",
					"description": "Content to write to the file",
				},
			},
			"required": []string{"path", "content"},
		},
	}, func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args struct {
			Path    string `json:"path"`
			Content string `json:"content"`
		}
		if err := json.Unmarshal(request.Params.Arguments, &args); err != nil {
			return errorResult(fmt.Sprintf("Invalid arguments: %v", err)), nil
		}

		validPath, err := s.validatePath(args.Path)
		if err != nil {
			return errorResult(err.Error()), nil
		}

		err = os.WriteFile(validPath, []byte(args.Content), 0644)
		if err != nil {
			return errorResult(fmt.Sprintf("Error writing file: %v", err)), nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{
				Text: fmt.Sprintf("Successfully wrote to %s", args.Path),
			}},
		}, nil
	})

	s.server.AddTool(&mcp.Tool{
		Name: "move_file",
		Description: `Move or rename files and directories. Fails if destination already exists.`,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"source": map[string]any{
					"type": "string",
					"description": "Source path",
				},
				"destination": map[string]any{
					"type": "string",
					"description": "Destination path",
				},
			},
			"required": []string{"source", "destination"},
		},
	}, func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args struct {
			Source      string `json:"source"`
			Destination string `json:"destination"`
		}
		if err := json.Unmarshal(request.Params.Arguments, &args); err != nil {
			return errorResult(fmt.Sprintf("Invalid arguments: %v", err)), nil
		}

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

		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{
				Text: fmt.Sprintf("Successfully moved %s to %s", args.Source, args.Destination),
			}},
		}, nil
	})
}
