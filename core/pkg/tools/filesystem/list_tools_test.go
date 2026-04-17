package filesystem

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func (s *FilesystemTestSuite) TestListDirectory() {
	s.createFile("file1.txt", "")
	s.createFile("file2.txt", "")
	err := os.MkdirAll(filepath.Join(s.testDir, "subdir"), 0755)
	s.Require().NoError(err)

	args, _ := json.Marshal(map[string]any{"path": s.testDir})
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "list_directory",
			Arguments: args,
		},
	}

	res, err := s.fsServer.handleListDirectory(s.ctx, req)
	s.Require().NoError(err)
	s.Require().False(res.IsError)

	output := res.Content[0].(*mcp.TextContent).Text
	s.Contains(output, "[FILE] file1.txt")
	s.Contains(output, "[FILE] file2.txt")
	s.Contains(output, "[DIR] subdir")

	// Test outside path
	args, _ = json.Marshal(map[string]any{"path": "/etc"})
	req.Params.Arguments = args
	res, err = s.fsServer.handleListDirectory(s.ctx, req)
	s.Require().NoError(err)
	s.Require().True(res.IsError)

	// Test invalid JSON
	req.Params.Arguments = []byte(`{"path": 123}`)
	res, err = s.fsServer.handleListDirectory(s.ctx, req)
	s.Require().NoError(err)
	s.Require().True(res.IsError)

	// Test non-existent directory
	args, _ = json.Marshal(map[string]any{"path": filepath.Join(s.testDir, "missing")})
	req.Params.Arguments = args
	res, err = s.fsServer.handleListDirectory(s.ctx, req)
	s.Require().NoError(err)
	s.Require().True(res.IsError)
}

func (s *FilesystemTestSuite) TestListDirectoryWithSizes() {
	s.createFile("small.txt", "abc")    // 3 bytes
	s.createFile("large.txt", "abcdef") // 6 bytes
	err := os.MkdirAll(filepath.Join(s.testDir, "subdir"), 0755)
	s.Require().NoError(err)

	// Sort by name with sizes
	args, _ := json.Marshal(map[string]any{"path": s.testDir, "include_sizes": true, "sortBy": "name"})
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "list_directory",
			Arguments: args,
		},
	}

	res, err := s.fsServer.handleListDirectory(s.ctx, req)
	s.Require().NoError(err)
	s.Require().False(res.IsError)

	output := res.Content[0].(*mcp.TextContent).Text
	s.Contains(output, "[FILE] large.txt (6 bytes)")
	s.Contains(output, "[FILE] small.txt (3 bytes)")
	s.Contains(output, "[DIR] subdir")
	s.Contains(output, "Summary: 2 files, 1 directories, 9 bytes total")

	// Sort by size
	args, _ = json.Marshal(map[string]any{"path": s.testDir, "include_sizes": true, "sortBy": "size"})
	req.Params.Arguments = args
	res, err = s.fsServer.handleListDirectory(s.ctx, req)
	s.Require().NoError(err)
	s.Require().False(res.IsError)
	output = res.Content[0].(*mcp.TextContent).Text

	largeIdx := -1
	smallIdx := -1
	lines := []string{"[FILE] large.txt (6 bytes)", "[FILE] small.txt (3 bytes)"}
	// Verify large.txt appears before small.txt in the output
	for i := range output {
		if output[i:min(i+len(lines[0]), len(output))] == lines[0] {
			largeIdx = i
		}
		if output[i:min(i+len(lines[1]), len(output))] == lines[1] {
			smallIdx = i
		}
	}
	s.True(largeIdx < smallIdx)

	// Test outside path
	args, _ = json.Marshal(map[string]any{"path": "/etc", "include_sizes": true})
	req.Params.Arguments = args
	res, err = s.fsServer.handleListDirectory(s.ctx, req)
	s.Require().NoError(err)
	s.Require().True(res.IsError)
}

func (s *FilesystemTestSuite) TestSearchFiles() {
	s.createFile("src/main.go", "main")
	s.createFile("src/utils.go", "utils")
	s.createFile("test/main_test.go", "test")
	s.createFile("README.md", "readme")
	s.createFile("vendor/ignored.go", "vendor")

	// Search for *.go
	args, _ := json.Marshal(map[string]any{
		"path":            s.testDir,
		"pattern":         "**/*.go",
		"excludePatterns": []string{"**/vendor/**"},
	})
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "search_files",
			Arguments: args,
		},
	}

	res, err := s.fsServer.handleSearchFiles(s.ctx, req)
	s.Require().NoError(err)
	s.Require().False(res.IsError)

	output := res.Content[0].(*mcp.TextContent).Text
	s.Contains(output, "main.go")
	s.Contains(output, "utils.go")
	s.Contains(output, "main_test.go")
	s.NotContains(output, "README.md")
	s.NotContains(output, "ignored.go")

	// Test invalid JSON
	req.Params.Arguments = []byte(`{"path": 123}`)
	res, err = s.fsServer.handleSearchFiles(s.ctx, req)
	s.Require().NoError(err)
	s.Require().True(res.IsError)

	// Test outside path
	args, _ = json.Marshal(map[string]any{
		"path":    "/etc",
		"pattern": "*",
	})
	req.Params.Arguments = args
	res, err = s.fsServer.handleSearchFiles(s.ctx, req)
	s.Require().NoError(err)
	s.Require().True(res.IsError)
}

func (s *FilesystemTestSuite) TestDirectoryTree() {
	s.createFile("src/main.go", "")
	s.createFile("docs/readme.md", "")
	s.createFile("vendor/dep/dep.go", "")

	args, _ := json.Marshal(map[string]any{
		"path":            s.testDir,
		"excludePatterns": []string{"**/vendor"},
	})
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "directory_tree",
			Arguments: args,
		},
	}

	res, err := s.fsServer.handleDirectoryTree(s.ctx, req)
	s.Require().NoError(err)
	s.Require().False(res.IsError)

	var tree []*Node
	err = json.Unmarshal([]byte(res.Content[0].(*mcp.TextContent).Text), &tree)
	s.Require().NoError(err)

	// Tree should have src and docs, but not vendor
	var foundSrc, foundDocs, foundVendor bool
	for _, node := range tree {
		if node.Name == "src" {
			foundSrc = true
		}
		if node.Name == "docs" {
			foundDocs = true
		}
		if node.Name == "vendor" {
			foundVendor = true
		}
	}
	s.True(foundSrc)
	s.True(foundDocs)
	s.False(foundVendor)

	// Test invalid path
	args, _ = json.Marshal(map[string]any{"path": "/etc"})
	req.Params.Arguments = args
	res, err = s.fsServer.handleDirectoryTree(s.ctx, req)
	s.Require().NoError(err)
	s.Require().True(res.IsError)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (s *FilesystemTestSuite) TestListDirectoryJSONFormat() {
	s.createFile("file1.txt", "")
	s.createFile("file2.txt", "")
	err := os.MkdirAll(filepath.Join(s.testDir, "subdir"), 0755)
	s.Require().NoError(err)

	args, _ := json.Marshal(map[string]any{"path": s.testDir, "format": "json"})
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "list_directory",
			Arguments: args,
		},
	}

	res, err := s.fsServer.handleListDirectory(s.ctx, req)
	s.Require().NoError(err)
	s.Require().False(res.IsError)

	var result ListDirectoryResult
	err = json.Unmarshal([]byte(res.Content[0].(*mcp.TextContent).Text), &result)
	s.Require().NoError(err)
	s.Equal(s.testDir, result.Path)
	s.Equal(3, len(result.Entries))
	s.Equal(2, result.Summary.Files)
	s.Equal(1, result.Summary.Directories)
}

func (s *FilesystemTestSuite) TestListDirectoryWithSizesJSONFormat() {
	s.createFile("small.txt", "abc")
	s.createFile("large.txt", "abcdef")

	args, _ := json.Marshal(map[string]any{"path": s.testDir, "include_sizes": true, "format": "json", "sortBy": "size"})
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "list_directory",
			Arguments: args,
		},
	}

	res, err := s.fsServer.handleListDirectory(s.ctx, req)
	s.Require().NoError(err)
	s.Require().False(res.IsError)

	var result ListDirectoryResult
	err = json.Unmarshal([]byte(res.Content[0].(*mcp.TextContent).Text), &result)
	s.Require().NoError(err)
	s.Equal(2, result.Summary.Files)
	s.Equal(int64(9), result.Summary.TotalSize)
}

func (s *FilesystemTestSuite) TestSearchFilesJSONFormat() {
	s.createFile("src/main.go", "main")
	s.createFile("src/utils.go", "utils")

	args, _ := json.Marshal(map[string]any{
		"path":    s.testDir,
		"pattern": "**/*.go",
		"format":  "json",
	})
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "search_files",
			Arguments: args,
		},
	}

	res, err := s.fsServer.handleSearchFiles(s.ctx, req)
	s.Require().NoError(err)
	s.Require().False(res.IsError)

	var result SearchResult
	err = json.Unmarshal([]byte(res.Content[0].(*mcp.TextContent).Text), &result)
	s.Require().NoError(err)
	s.Equal(2, result.Summary.TotalMatches)
	s.Equal("**/*.go", result.Pattern)
}

func (s *FilesystemTestSuite) TestDirectoryTreeJSONFormat() {
	s.createFile("src/main.go", "")
	s.createFile("docs/readme.md", "")

	args, _ := json.Marshal(map[string]any{
		"path":   s.testDir,
		"format": "json",
	})
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "directory_tree",
			Arguments: args,
		},
	}

	res, err := s.fsServer.handleDirectoryTree(s.ctx, req)
	s.Require().NoError(err)
	s.Require().False(res.IsError)

	var result DirectoryTreeResult
	err = json.Unmarshal([]byte(res.Content[0].(*mcp.TextContent).Text), &result)
	s.Require().NoError(err)
	s.NotNil(result.Root)
	s.Equal(2, result.Summary.Files)
	s.Equal(3, result.Summary.Directories) // testDir + src + docs
}
