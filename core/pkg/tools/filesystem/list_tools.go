package filesystem

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func (s *FilesystemServer) registerListTools() {
	s.server.AddTool(&mcp.Tool{
		Name: "list_directory",
		Description: `List directory contents with [FILE] or [DIR] prefixes.`,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type": "string",
					"description": "Path to the directory to list",
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

		entries, err := os.ReadDir(validPath)
		if err != nil {
			return errorResult(fmt.Sprintf("Error reading directory: %v", err)), nil
		}

		var lines []string
		for _, entry := range entries {
			prefix := "[FILE]"
			if entry.IsDir() {
				prefix := "[DIR]"
				lines = append(lines, fmt.Sprintf("%s %s", prefix, entry.Name()))
			} else {
				lines = append(lines, fmt.Sprintf("%s %s", prefix, entry.Name()))
			}
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{
				Text: strings.Join(lines, "\n"),
			}},
		}, nil
	})

	s.server.AddTool(&mcp.Tool{
		Name: "list_directory_with_sizes",
		Description: `List directory contents with [FILE] or [DIR] prefixes, including file sizes.`,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type": "string",
					"description": "Directory path to list",
				},
				"sortBy": map[string]any{
					"type": "string",
					"description": "Sort entries by 'name' or 'size' (default: 'name')",
					"enum": []string{"name", "size"},
				},
			},
			"required": []string{"path"},
		},
	}, func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args struct {
			Path   string `json:"path"`
			SortBy string `json:"sortBy"`
		}
		if err := json.Unmarshal(request.Params.Arguments, &args); err != nil {
			return errorResult(fmt.Sprintf("Invalid arguments: %v", err)), nil
		}

		if args.SortBy == "" {
			args.SortBy = "name"
		}

		validPath, err := s.validatePath(args.Path)
		if err != nil {
			return errorResult(err.Error()), nil
		}

		entries, err := os.ReadDir(validPath)
		if err != nil {
			return errorResult(fmt.Sprintf("Error reading directory: %v", err)), nil
		}

		type dirEntry struct {
			Name  string
			IsDir bool
			Size  int64
		}

		var dirEntries []dirEntry
		var totalFiles, totalDirs int
		var totalSize int64

		for _, entry := range entries {
			info, err := entry.Info()
			var size int64
			if err == nil {
				size = info.Size()
			}
			
			dirEntries = append(dirEntries, dirEntry{
				Name:  entry.Name(),
				IsDir: entry.IsDir(),
				Size:  size,
			})

			if entry.IsDir() {
				totalDirs++
			} else {
				totalFiles++
				totalSize += size
			}
		}

		if args.SortBy == "size" {
			sort.Slice(dirEntries, func(i, j int) bool {
				if dirEntries[i].Size == dirEntries[j].Size {
					return dirEntries[i].Name < dirEntries[j].Name
				}
				return dirEntries[i].Size > dirEntries[j].Size // Descending
			})
		} else {
			sort.Slice(dirEntries, func(i, j int) bool {
				return dirEntries[i].Name < dirEntries[j].Name
			})
		}

		var lines []string
		for _, e := range dirEntries {
			prefix := "[FILE]"
			if e.IsDir {
				prefix = "[DIR]"
				lines = append(lines, fmt.Sprintf("%s %s", prefix, e.Name))
			} else {
				lines = append(lines, fmt.Sprintf("%s %s (%d bytes)", prefix, e.Name, e.Size))
			}
		}

		lines = append(lines, fmt.Sprintf("\nSummary: %d files, %d directories, %d bytes total", totalFiles, totalDirs, totalSize))

		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{
				Text: strings.Join(lines, "\n"),
			}},
		}, nil
	})

	s.server.AddTool(&mcp.Tool{
		Name: "search_files",
		Description: `Recursively search for files/directories that match a pattern.`,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type": "string",
					"description": "Starting directory",
				},
				"pattern": map[string]any{
					"type": "string",
					"description": "Search pattern (glob)",
				},
				"excludePatterns": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "string",
					},
					"description": "Exclude patterns",
				},
			},
			"required": []string{"path", "pattern"},
		},
	}, func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args struct {
			Path            string   `json:"path"`
			Pattern         string   `json:"pattern"`
			ExcludePatterns []string `json:"excludePatterns"`
		}
		if err := json.Unmarshal(request.Params.Arguments, &args); err != nil {
			return errorResult(fmt.Sprintf("Invalid arguments: %v", err)), nil
		}

		validPath, err := s.validatePath(args.Path)
		if err != nil {
			return errorResult(err.Error()), nil
		}

		fileSys := os.DirFS(validPath)
		var matches []string

		err = fs.WalkDir(fileSys, ".", func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil // Skip errors
			}

			// Exclude patterns
			for _, exclude := range args.ExcludePatterns {
				matched, err := doublestar.Match(exclude, path)
				if err == nil && matched {
					if d.IsDir() {
						return fs.SkipDir
					}
					return nil
				}
			}

			matched, err := doublestar.Match(args.Pattern, path)
			if err == nil && matched {
				fullPath := filepath.Join(validPath, filepath.FromSlash(path))
				matches = append(matches, fullPath)
			}

			return nil
		})

		if err != nil {
			return errorResult(fmt.Sprintf("Search error: %v", err)), nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{
				Text: strings.Join(matches, "\n"),
			}},
		}, nil
	})

	s.server.AddTool(&mcp.Tool{
		Name: "directory_tree",
		Description: `Get recursive JSON tree structure of directory contents.`,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type": "string",
					"description": "Starting directory",
				},
				"excludePatterns": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "string",
					},
					"description": "Exclude patterns",
				},
			},
			"required": []string{"path"},
		},
	}, func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args struct {
			Path            string   `json:"path"`
			ExcludePatterns []string `json:"excludePatterns"`
		}
		if err := json.Unmarshal(request.Params.Arguments, &args); err != nil {
			return errorResult(fmt.Sprintf("Invalid arguments: %v", err)), nil
		}

		validPath, err := s.validatePath(args.Path)
		if err != nil {
			return errorResult(err.Error()), nil
		}

		type Node struct {
			Name     string  `json:"name"`
			Type     string  `json:"type"`
			Children []*Node `json:"children,omitempty"`
		}

		var buildTree func(currentPath string, relativePath string) (*Node, error)
		buildTree = func(currentPath string, relativePath string) (*Node, error) {
			info, err := os.Stat(currentPath)
			if err != nil {
				return nil, err
			}

			node := &Node{
				Name: info.Name(),
				Type: "file",
			}

			if info.IsDir() {
				node.Type = "directory"
				node.Children = make([]*Node, 0)

				entries, err := os.ReadDir(currentPath)
				if err != nil {
					return node, nil // Ignore read errors for individual dirs
				}

				for _, entry := range entries {
					childRelPath := filepath.ToSlash(filepath.Join(relativePath, entry.Name()))
					
					excluded := false
					for _, exclude := range args.ExcludePatterns {
						matched, _ := doublestar.Match(exclude, childRelPath)
						if matched {
							excluded = true
							break
						}
					}
					if excluded {
						continue
					}

					childPath := filepath.Join(currentPath, entry.Name())
					childNode, err := buildTree(childPath, childRelPath)
					if err == nil && childNode != nil {
						node.Children = append(node.Children, childNode)
					}
				}
			}

			return node, nil
		}

		rootNode, err := buildTree(validPath, ".")
		if err != nil {
			return errorResult(fmt.Sprintf("Error building tree: %v", err)), nil
		}

		// The root node is usually a directory, but the TypeScript server returns an array of its children
		var result []*Node
		if rootNode.Type == "directory" {
			result = rootNode.Children
		} else {
			result = []*Node{rootNode}
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
	})
}
