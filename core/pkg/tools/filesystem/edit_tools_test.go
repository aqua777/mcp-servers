package filesystem

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func (s *FilesystemTestSuite) TestEditFile() {
	testFile := s.createFile("edit_me.txt", "line1\nline2\nline3\n")

	// Test successful edit
	args, _ := json.Marshal(map[string]any{
		"path": testFile,
		"edits": []map[string]string{
			{"oldText": "line2", "newText": "line2_edited"},
		},
		"dryRun": false,
	})
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "edit_file",
			Arguments: args,
		},
	}

	res, err := s.fsServer.handleEditFile(s.ctx, req)
	s.Require().NoError(err)
	s.Require().False(res.IsError)
	s.Contains(res.Content[0].(*mcp.TextContent).Text, "Successfully edited")

	content, err := os.ReadFile(testFile)
	s.Require().NoError(err)
	s.Equal("line1\nline2_edited\nline3\n", string(content))

	// Test dry run
	args, _ = json.Marshal(map[string]any{
		"path": testFile,
		"edits": []map[string]string{
			{"oldText": "line3", "newText": "line3_edited"},
		},
		"dryRun": true,
	})
	req.Params.Arguments = args
	res, err = s.fsServer.handleEditFile(s.ctx, req)
	s.Require().NoError(err)
	s.Require().False(res.IsError)
	s.Contains(res.Content[0].(*mcp.TextContent).Text, "Dry run diff")

	// Verify file was NOT modified in dry run
	content, err = os.ReadFile(testFile)
	s.Require().NoError(err)
	s.Equal("line1\nline2_edited\nline3\n", string(content))

	// Test old text not found
	args, _ = json.Marshal(map[string]any{
		"path": testFile,
		"edits": []map[string]string{
			{"oldText": "missing", "newText": "new"},
		},
	})
	req.Params.Arguments = args
	res, err = s.fsServer.handleEditFile(s.ctx, req)
	s.Require().NoError(err)
	s.Require().True(res.IsError)
	s.Contains(res.Content[0].(*mcp.TextContent).Text, "oldText not found")

	// Test invalid path
	args, _ = json.Marshal(map[string]any{
		"path": "/etc/passwd",
		"edits": []map[string]string{{"oldText": "", "newText": ""}},
	})
	req.Params.Arguments = args
	res, err = s.fsServer.handleEditFile(s.ctx, req)
	s.Require().NoError(err)
	s.Require().True(res.IsError)
}

func (s *FilesystemTestSuite) TestGetFileInfo() {
	testFile := s.createFile("info.txt", "some data")

	args, _ := json.Marshal(map[string]any{"path": testFile})
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "get_file_info",
			Arguments: args,
		},
	}

	res, err := s.fsServer.handleGetFileInfo(s.ctx, req)
	s.Require().NoError(err)
	s.Require().False(res.IsError)

	var info map[string]any
	err = json.Unmarshal([]byte(res.Content[0].(*mcp.TextContent).Text), &info)
	s.Require().NoError(err)
	
	s.Equal("file", info["type"])
	s.Equal(float64(9), info["size"]) // JSON unmarshals numbers as float64

	// Test directory info
	args, _ = json.Marshal(map[string]any{"path": s.testDir})
	req.Params.Arguments = args
	res, err = s.fsServer.handleGetFileInfo(s.ctx, req)
	s.Require().NoError(err)
	s.Require().False(res.IsError)
	
	err = json.Unmarshal([]byte(res.Content[0].(*mcp.TextContent).Text), &info)
	s.Require().NoError(err)
	s.Equal("directory", info["type"])

	// Test invalid path
	args, _ = json.Marshal(map[string]any{"path": "/etc/passwd"})
	req.Params.Arguments = args
	res, err = s.fsServer.handleGetFileInfo(s.ctx, req)
	s.Require().NoError(err)
	s.Require().True(res.IsError)
	
	// Test file not found
	args, _ = json.Marshal(map[string]any{"path": filepath.Join(s.testDir, "missing.txt")})
	req.Params.Arguments = args
	res, err = s.fsServer.handleGetFileInfo(s.ctx, req)
	s.Require().NoError(err)
	s.Require().True(res.IsError)
}

func (s *FilesystemTestSuite) TestListAllowedDirectories() {
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "list_allowed_directories",
			Arguments: []byte(`{}`),
		},
	}

	res, err := s.fsServer.handleListAllowedDirectories(s.ctx, req)
	s.Require().NoError(err)
	s.Require().False(res.IsError)

	var result map[string][]string
	err = json.Unmarshal([]byte(res.Content[0].(*mcp.TextContent).Text), &result)
	s.Require().NoError(err)
	
	s.Len(result["allowedDirectories"], 1)
	s.Equal(s.allowedDir, result["allowedDirectories"][0])
}
