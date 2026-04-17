package filesystem

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func (s *FilesystemServer) registerListTools() {
	s.server.AddTool(&mcp.Tool{
		Name:        "list_directory",
		Description: `List directory contents with [FILE] or [DIR] prefixes. Optionally include file sizes and sort by name or size.`,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "Path to the directory to list",
				},
				"include_sizes": map[string]any{
					"type":        "boolean",
					"description": "Include file sizes in output (default: false)",
					"default":     false,
				},
				"sortBy": map[string]any{
					"type":        "string",
					"description": "Sort entries by 'name' or 'size' (default: 'name')",
					"enum":        []string{"name", "size"},
				},
				"format": map[string]any{
					"type":        "string",
					"enum":        []string{"text", "json"},
					"description": "Output format (default: server setting)",
				},
			},
			"required": []string{"path"},
		},
	}, s.handleListDirectory)

	s.server.AddTool(&mcp.Tool{
		Name:        "search_files",
		Description: `Recursively search for files/directories that match a pattern.`,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "Starting directory",
				},
				"pattern": map[string]any{
					"type":        "string",
					"description": "Search pattern (glob)",
				},
				"excludePatterns": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "string",
					},
					"description": "Exclude patterns",
				},
				"format": map[string]any{
					"type":        "string",
					"enum":        []string{"text", "json"},
					"description": "Output format (default: server setting)",
				},
			},
			"required": []string{"path", "pattern"},
		},
	}, s.handleSearchFiles)

	s.server.AddTool(&mcp.Tool{
		Name:        "directory_tree",
		Description: `Get recursive JSON tree structure of directory contents.`,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "Starting directory",
				},
				"excludePatterns": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "string",
					},
					"description": "Exclude patterns",
				},
				"format": map[string]any{
					"type":        "string",
					"enum":        []string{"text", "json"},
					"description": "Output format (default: server setting)",
				},
			},
			"required": []string{"path"},
		},
	}, s.handleDirectoryTree)
}

func (s *FilesystemServer) handleListDirectory(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args struct {
		Path         string `json:"path"`
		IncludeSizes bool   `json:"include_sizes"`
		SortBy       string `json:"sortBy"`
		Format       string `json:"format"`
	}
	if err := json.Unmarshal(request.Params.Arguments, &args); err != nil {
		return errorResult(fmt.Sprintf("Invalid arguments: %v", err)), nil
	}

	format := s.resolveFormat(args.Format)
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
		var size int64
		if args.IncludeSizes {
			info, err := entry.Info()
			if err == nil {
				size = info.Size()
			}
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

	// Sort entries
	if args.SortBy == "size" {
		sort.Slice(dirEntries, func(i, j int) bool {
			if dirEntries[i].Size == dirEntries[j].Size {
				return dirEntries[i].Name < dirEntries[j].Name
			}
			return dirEntries[i].Size > dirEntries[j].Size
		})
	} else {
		sort.Slice(dirEntries, func(i, j int) bool {
			return dirEntries[i].Name < dirEntries[j].Name
		})
	}

	// Build structured result
	var structuredEntries []DirectoryEntry
	for _, e := range dirEntries {
		entryType := "file"
		if e.IsDir {
			entryType = "directory"
		}
		var sizePtr *int64
		if args.IncludeSizes {
			sizePtr = &e.Size
		}
		structuredEntries = append(structuredEntries, DirectoryEntry{
			Name: e.Name,
			Type: entryType,
			Size: sizePtr,
		})
	}

	result := &ListDirectoryResult{
		Path:    validPath,
		Entries: structuredEntries,
		Summary: DirectorySummary{
			Files:       totalFiles,
			Directories: totalDirs,
			TotalSize:   totalSize,
		},
		SortBy: args.SortBy,
	}

	var text string
	if format == FormatJSON {
		text = formatListDirectoryJSON(result)
	} else {
		text = formatListDirectoryText(result)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{
			Text: text,
		}},
	}, nil
}

func (s *FilesystemServer) handleSearchFiles(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args struct {
		Path            string   `json:"path"`
		Pattern         string   `json:"pattern"`
		ExcludePatterns []string `json:"excludePatterns"`
		Format          string   `json:"format"`
	}
	if err := json.Unmarshal(request.Params.Arguments, &args); err != nil {
		return errorResult(fmt.Sprintf("Invalid arguments: %v", err)), nil
	}

	format := s.resolveFormat(args.Format)
	validPath, err := s.validatePath(args.Path)
	if err != nil {
		return errorResult(err.Error()), nil
	}

	fileSys := os.DirFS(validPath)
	var matches []SearchMatch

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
			entryType := "file"
			if d.IsDir() {
				entryType = "directory"
			}
			matches = append(matches, SearchMatch{
				Path: fullPath,
				Type: entryType,
			})
		}

		return nil
	})

	if err != nil {
		return errorResult(fmt.Sprintf("Search error: %v", err)), nil
	}

	result := &SearchResult{
		Root:    validPath,
		Pattern: args.Pattern,
		Matches: matches,
		Summary: SearchSummary{
			TotalMatches: len(matches),
		},
	}

	var text string
	if format == FormatJSON {
		text = formatSearchFilesJSON(result)
	} else {
		text = formatSearchFilesText(result)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{
			Text: text,
		}},
	}, nil
}

type Node struct {
	Name     string  `json:"name"`
	Type     string  `json:"type"`
	Children []*Node `json:"children,omitempty"`
}

func (s *FilesystemServer) handleDirectoryTree(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args struct {
		Path            string   `json:"path"`
		ExcludePatterns []string `json:"excludePatterns"`
		Format          string   `json:"format"`
	}
	if err := json.Unmarshal(request.Params.Arguments, &args); err != nil {
		return errorResult(fmt.Sprintf("Invalid arguments: %v", err)), nil
	}

	format := s.resolveFormat(args.Format)
	validPath, err := s.validatePath(args.Path)
	if err != nil {
		return errorResult(err.Error()), nil
	}

	var totalFiles, totalDirs int
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
			totalDirs++

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
		} else {
			totalFiles++
		}

		return node, nil
	}

	rootNode, err := buildTree(validPath, ".")
	if err != nil {
		return errorResult(fmt.Sprintf("Error building tree: %v", err)), nil
	}

	// The root node is usually a directory, but the TypeScript server returns an array of its children
	var rootForResult *Node
	if rootNode.Type == "directory" {
		rootForResult = &Node{
			Name:     rootNode.Name,
			Type:     "directory",
			Children: rootNode.Children,
		}
	} else {
		rootForResult = rootNode
	}

	result := &DirectoryTreeResult{
		Root: rootForResult,
		Summary: DirectorySummary{
			Files:       totalFiles,
			Directories: totalDirs,
		},
	}

	var text string
	if format == FormatJSON {
		text = formatDirectoryTreeJSON(result)
	} else {
		text = formatDirectoryTreeText(result)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{
			Text: text,
		}},
	}, nil
}
