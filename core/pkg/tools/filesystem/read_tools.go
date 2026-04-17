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
		Description: `Read the complete contents of a file as text. Always treats the file as UTF-8 text regardless of extension. Use 'head' to read only the first N lines, or 'tail' to read only the last N lines.`,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "Path to the file to read",
				},
				"head": map[string]any{
					"type":        "integer",
					"description": "If provided, returns only the first N lines of the file",
				},
				"tail": map[string]any{
					"type":        "integer",
					"description": "If provided, returns only the last N lines of the file",
				},
				"format": map[string]any{
					"type":        "string",
					"enum":        []string{"text", "json"},
					"description": "Output format (default: server setting)",
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
				"format": map[string]any{
					"type":        "string",
					"enum":        []string{"text", "json"},
					"description": "Output format (default: server setting)",
				},
			},
			"required": []string{"paths"},
		},
	}, s.handleReadMultipleFiles)
}

func (s *FilesystemServer) handleReadTextFile(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args struct {
		Path   string `json:"path"`
		Head   *int   `json:"head"`
		Tail   *int   `json:"tail"`
		Format string `json:"format"`
	}
	if err := json.Unmarshal(request.Params.Arguments, &args); err != nil {
		return errorResult(fmt.Sprintf("Invalid arguments: %v", err)), nil
	}

	format := s.resolveFormat(args.Format)
	if args.Head != nil && args.Tail != nil {
		return errorResult("Cannot specify both head and tail parameters simultaneously"), nil
	}

	validPath, err := s.validatePath(args.Path)
	if err != nil {
		return errorResult(err.Error()), nil
	}

	content, err := os.ReadFile(validPath)
	if err != nil {
		return errorResult(fmt.Sprintf("Error reading file: %v", err)), nil
	}

	text := string(content)
	originalSize := int64(len(content))

	if args.Head != nil || args.Tail != nil {
		lines := strings.Split(text, "\n")
		if args.Head != nil {
			n := *args.Head
			if n < 0 {
				return errorResult("head parameter must be non-negative"), nil
			}
			if n < len(lines) {
				lines = lines[:n]
			}
		} else if args.Tail != nil {
			n := *args.Tail
			if n < 0 {
				return errorResult("tail parameter must be non-negative"), nil
			}
			if n < len(lines) {
				lines = lines[len(lines)-n:]
			}
		}
		text = strings.Join(lines, "\n")
	}

	result := &ReadFileResult{
		Path:    validPath,
		Size:    originalSize,
		Lines:   len(strings.Split(text, "\n")),
		Content: text,
	}

	var output string
	if format == FormatJSON {
		output = formatReadFileJSON(result)
	} else {
		output = formatReadFileText(result)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{
			Text: output,
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
		Paths  []string `json:"paths"`
		Format string   `json:"format"`
	}
	if err := json.Unmarshal(request.Params.Arguments, &args); err != nil {
		return errorResult(fmt.Sprintf("Invalid arguments: %v", err)), nil
	}

	format := s.resolveFormat(args.Format)
	var fileResults []FileReadResult
	var succeeded, failed int

	for _, p := range args.Paths {
		validPath, err := s.validatePath(p)
		if err != nil {
			fileResults = append(fileResults, FileReadResult{
				Path:   p,
				Status: "error",
				Error:  err.Error(),
			})
			failed++
			continue
		}

		content, err := os.ReadFile(validPath)
		if err != nil {
			fileResults = append(fileResults, FileReadResult{
				Path:   p,
				Status: "error",
				Error:  fmt.Sprintf("Error reading file: %v", err),
			})
			failed++
		} else {
			fileResults = append(fileResults, FileReadResult{
				Path:    p,
				Status:  "ok",
				Content: string(content),
			})
			succeeded++
		}
	}

	result := &ReadMultipleFilesResult{
		Files: fileResults,
		Summary: ReadMultipleFilesSummary{
			Succeeded: succeeded,
			Failed:    failed,
			Total:     len(args.Paths),
		},
	}

	var text string
	if format == FormatJSON {
		text = formatReadMultipleFilesJSON(result)
	} else {
		text = formatReadMultipleFilesText(result)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{
			Text: text,
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
