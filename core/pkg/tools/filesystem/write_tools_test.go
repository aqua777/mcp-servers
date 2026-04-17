package filesystem

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func (s *FilesystemTestSuite) TestCreateDirectory() {
	newDir := filepath.Join(s.testDir, "new_folder")

	args, _ := json.Marshal(map[string]any{"path": newDir})
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "create_directory",
			Arguments: args,
		},
	}

	// Test successful creation
	res, err := s.fsServer.handleCreateDirectory(s.ctx, req)
	s.Require().NoError(err)
	s.Require().False(res.IsError)
	s.Contains(res.Content[0].(*mcp.TextContent).Text, "Successfully created directory")

	// Verify directory was actually created
	info, err := os.Stat(newDir)
	s.Require().NoError(err)
	s.True(info.IsDir())

	// Test outside path
	req.Params.Arguments = []byte(`{"path": "/etc/invalid_dir"}`)
	res, err = s.fsServer.handleCreateDirectory(s.ctx, req)
	s.Require().NoError(err)
	s.Require().True(res.IsError)

	// Test invalid JSON arguments
	req.Params.Arguments = []byte(`{"path": 123}`)
	res, err = s.fsServer.handleCreateDirectory(s.ctx, req)
	s.Require().NoError(err)
	s.Require().True(res.IsError)
}

func (s *FilesystemTestSuite) TestWriteFile() {
	newFile := filepath.Join(s.testDir, "test_write.txt")
	content := "Hello, Write File!"

	args, _ := json.Marshal(map[string]any{
		"path":    newFile,
		"content": content,
	})
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "write_file",
			Arguments: args,
		},
	}

	// Test successful write
	res, err := s.fsServer.handleWriteFile(s.ctx, req)
	s.Require().NoError(err)
	s.Require().False(res.IsError)
	s.Contains(res.Content[0].(*mcp.TextContent).Text, "Successfully wrote")

	// Verify file content
	writtenContent, err := os.ReadFile(newFile)
	s.Require().NoError(err)
	s.Equal(content, string(writtenContent))

	// Test outside path
	args, _ = json.Marshal(map[string]any{
		"path":    "/etc/passwd",
		"content": "hacked",
	})
	req.Params.Arguments = args
	res, err = s.fsServer.handleWriteFile(s.ctx, req)
	s.Require().NoError(err)
	s.Require().True(res.IsError)

	// Test invalid JSON arguments
	req.Params.Arguments = []byte(`{"path": 123}`)
	res, err = s.fsServer.handleWriteFile(s.ctx, req)
	s.Require().NoError(err)
	s.Require().True(res.IsError)
}

func (s *FilesystemTestSuite) TestMoveFile() {
	srcFile := s.createFile("src.txt", "move me")
	destFile := filepath.Join(s.testDir, "dest.txt")

	args, _ := json.Marshal(map[string]any{
		"source":      srcFile,
		"destination": destFile,
	})
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "move_file",
			Arguments: args,
		},
	}

	// Test successful move
	res, err := s.fsServer.handleMoveFile(s.ctx, req)
	s.Require().NoError(err)
	s.Require().False(res.IsError)
	s.Contains(res.Content[0].(*mcp.TextContent).Text, "Successfully moved")

	// Verify move occurred
	_, err = os.Stat(srcFile)
	s.True(os.IsNotExist(err)) // Source should no longer exist

	content, err := os.ReadFile(destFile)
	s.Require().NoError(err)
	s.Equal("move me", string(content))

	// Test move to existing destination
	s.createFile("existing.txt", "I already exist")
	existingDest := filepath.Join(s.testDir, "existing.txt")
	srcFile2 := s.createFile("src2.txt", "move me 2")

	args, _ = json.Marshal(map[string]any{
		"source":      srcFile2,
		"destination": existingDest,
	})
	req.Params.Arguments = args
	res, err = s.fsServer.handleMoveFile(s.ctx, req)
	s.Require().NoError(err)
	s.Require().True(res.IsError)
	s.Contains(res.Content[0].(*mcp.TextContent).Text, "already exists")

	// Test invalid JSON arguments
	req.Params.Arguments = []byte(`{"source": 123}`)
	res, err = s.fsServer.handleMoveFile(s.ctx, req)
	s.Require().NoError(err)
	s.Require().True(res.IsError)

	// Test source outside path
	args, _ = json.Marshal(map[string]any{
		"source":      "/etc/passwd",
		"destination": destFile,
	})
	req.Params.Arguments = args
	res, err = s.fsServer.handleMoveFile(s.ctx, req)
	s.Require().NoError(err)
	s.Require().True(res.IsError)

	// Test destination outside path
	args, _ = json.Marshal(map[string]any{
		"source":      srcFile2,
		"destination": "/etc/passwd",
	})
	req.Params.Arguments = args
	res, err = s.fsServer.handleMoveFile(s.ctx, req)
	s.Require().NoError(err)
	s.Require().True(res.IsError)
}

func (s *FilesystemTestSuite) TestWriteFileJSONFormat() {
	newFile := filepath.Join(s.testDir, "test_write.txt")
	content := "Hello, Write File!"

	args, _ := json.Marshal(map[string]any{
		"path":    newFile,
		"content": content,
		"format":  "json",
	})
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "write_file",
			Arguments: args,
		},
	}

	res, err := s.fsServer.handleWriteFile(s.ctx, req)
	s.Require().NoError(err)
	s.Require().False(res.IsError)

	var result WriteResult
	err = json.Unmarshal([]byte(res.Content[0].(*mcp.TextContent).Text), &result)
	s.Require().NoError(err)
	s.Equal(newFile, result.Path)
	s.Equal("ok", result.Status)
	s.Equal(len(content), result.BytesWritten)
}

func (s *FilesystemTestSuite) TestCreateDirectoryJSONFormat() {
	newDir := filepath.Join(s.testDir, "new_folder")

	args, _ := json.Marshal(map[string]any{"path": newDir, "format": "json"})
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "create_directory",
			Arguments: args,
		},
	}

	res, err := s.fsServer.handleCreateDirectory(s.ctx, req)
	s.Require().NoError(err)
	s.Require().False(res.IsError)

	var result CreateDirectoryResult
	err = json.Unmarshal([]byte(res.Content[0].(*mcp.TextContent).Text), &result)
	s.Require().NoError(err)
	s.Equal(newDir, result.Path)
	s.True(result.Created)
}

func (s *FilesystemTestSuite) TestMoveFileJSONFormat() {
	srcFile := s.createFile("src.txt", "move me")
	destFile := filepath.Join(s.testDir, "dest.txt")

	args, _ := json.Marshal(map[string]any{
		"source":      srcFile,
		"destination": destFile,
		"format":      "json",
	})
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "move_file",
			Arguments: args,
		},
	}

	res, err := s.fsServer.handleMoveFile(s.ctx, req)
	s.Require().NoError(err)
	s.Require().False(res.IsError)

	var result MoveResult
	err = json.Unmarshal([]byte(res.Content[0].(*mcp.TextContent).Text), &result)
	s.Require().NoError(err)
	s.Equal(srcFile, result.Source)
	s.Equal(destFile, result.Destination)
	s.Equal("ok", result.Status)
}
