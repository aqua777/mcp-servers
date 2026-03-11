package filesystem

import (
	"context"
	"encoding/json"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func (s *FilesystemServer) registerReadTools() {
	s.server.AddTool(&mcp.Tool{
		Name:        "read_text_file",
		Description: `Read the complete contents of a file as text. Always treats the file as UTF-8 text regardless of extension.`,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "Path to the file to read",
				},
			},
			"required": []string{"path"},
		},
	}, s.handleReadTextFile)

	s.server.AddTool(&mcp.Tool{
		Name:        "read_media_file",
		Description: `Read an image or audio file and stream it as base64 with MIME type.`,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "Path to the media file to read",
				},
			},
			"required": []string{"path"},
		},
	}, s.handleReadMediaFile)

	s.server.AddTool(&mcp.Tool{
		Name:        "read_multiple_files",
		Description: `Read multiple files simultaneously. Failed reads won't stop the entire operation.`,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"paths": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "string",
					},
					"description": "Paths to the files to read",
				},
			},
			"required": []string{"paths"},
		},
	}, s.handleReadMultipleFiles)
}

func (s *FilesystemServer) handleReadTextFile(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

	content, err := os.ReadFile(validPath)
	if err != nil {
		return errorResult(fmt.Sprintf("Error reading file: %v", err)), nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{
			Text: string(content),
		}},
	}, nil
}

func (s *FilesystemServer) handleReadMediaFile(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

	content, err := os.ReadFile(validPath)
	if err != nil {
		return errorResult(fmt.Sprintf("Error reading file: %v", err)), nil
	}

	ext := filepath.Ext(validPath)
	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.ImageContent{
			Data:     content,
			MIMEType: mimeType,
		}},
	}, nil
}

func (s *FilesystemServer) handleReadMultipleFiles(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args struct {
		Paths []string `json:"paths"`
	}
	if err := json.Unmarshal(request.Params.Arguments, &args); err != nil {
		return errorResult(fmt.Sprintf("Invalid arguments: %v", err)), nil
	}

	var results []string
	for _, p := range args.Paths {
		validPath, err := s.validatePath(p)
		if err != nil {
			results = append(results, fmt.Sprintf("%s: Error: %v", p, err))
			continue
		}

		content, err := os.ReadFile(validPath)
		if err != nil {
			results = append(results, fmt.Sprintf("%s: Error reading file: %v", p, err))
		} else {
			results = append(results, fmt.Sprintf("--- %s ---\n%s", p, string(content)))
		}
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{
			Text: strings.Join(results, "\n\n"),
		}},
	}, nil
}

func errorResult(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{
			Text: msg,
		}},
		IsError: true,
	}
}
