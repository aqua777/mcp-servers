package filesystem

import (
	"encoding/json"
	"path/filepath"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func (s *FilesystemTestSuite) TestReadTextFile() {
	// Setup a test file
	testFile := s.createFile("hello.txt", "Hello, World!")

	// Test successful read
	args, _ := json.Marshal(map[string]any{"path": testFile})
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "read_text_file",
			Arguments: args,
		},
	}

	res, err := s.fsServer.handleReadTextFile(s.ctx, req)
	s.Require().NoError(err)
	s.Require().False(res.IsError)
	s.Require().Len(res.Content, 1)

	textContent, ok := res.Content[0].(*mcp.TextContent)
	s.Require().True(ok)
	s.Equal("Hello, World!", textContent.Text)

	// Test path outside allowed directory
	outsidePath := "/etc/passwd"
	args, _ = json.Marshal(map[string]any{"path": outsidePath})
	req.Params.Arguments = args

	res, err = s.fsServer.handleReadTextFile(s.ctx, req)
	s.Require().NoError(err)
	s.Require().True(res.IsError)
	errText := res.Content[0].(*mcp.TextContent).Text
	s.Contains(errText, "access denied")

	// Test file not found
	missingPath := filepath.Join(s.testDir, "missing.txt")
	args, _ = json.Marshal(map[string]any{"path": missingPath})
	req.Params.Arguments = args

	res, err = s.fsServer.handleReadTextFile(s.ctx, req)
	s.Require().NoError(err)
	s.Require().True(res.IsError)
	errText = res.Content[0].(*mcp.TextContent).Text
	s.Contains(errText, "Error reading file")

	// Test invalid JSON arguments
	req.Params.Arguments = []byte(`{"path": 123}`)
	res, err = s.fsServer.handleReadTextFile(s.ctx, req)
	s.Require().NoError(err)
	s.Require().True(res.IsError)
	s.Contains(res.Content[0].(*mcp.TextContent).Text, "Invalid arguments")
}

func (s *FilesystemTestSuite) TestReadTextFileWithHead() {
	testFile := s.createFile("lines.txt", "line1\nline2\nline3\nline4\nline5")

	// Test head parameter
	args, _ := json.Marshal(map[string]any{"path": testFile, "head": 3})
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "read_text_file",
			Arguments: args,
		},
	}

	res, err := s.fsServer.handleReadTextFile(s.ctx, req)
	s.Require().NoError(err)
	s.Require().False(res.IsError)
	s.Equal("line1\nline2\nline3", res.Content[0].(*mcp.TextContent).Text)

	// Test head exceeding file length
	args, _ = json.Marshal(map[string]any{"path": testFile, "head": 100})
	req.Params.Arguments = args
	res, err = s.fsServer.handleReadTextFile(s.ctx, req)
	s.Require().NoError(err)
	s.Require().False(res.IsError)
	s.Equal("line1\nline2\nline3\nline4\nline5", res.Content[0].(*mcp.TextContent).Text)

	// Test head = 0
	args, _ = json.Marshal(map[string]any{"path": testFile, "head": 0})
	req.Params.Arguments = args
	res, err = s.fsServer.handleReadTextFile(s.ctx, req)
	s.Require().NoError(err)
	s.Require().False(res.IsError)
	s.Equal("", res.Content[0].(*mcp.TextContent).Text)

	// Test negative head
	args, _ = json.Marshal(map[string]any{"path": testFile, "head": -1})
	req.Params.Arguments = args
	res, err = s.fsServer.handleReadTextFile(s.ctx, req)
	s.Require().NoError(err)
	s.Require().True(res.IsError)
	s.Contains(res.Content[0].(*mcp.TextContent).Text, "non-negative")
}

func (s *FilesystemTestSuite) TestReadTextFileWithTail() {
	testFile := s.createFile("lines.txt", "line1\nline2\nline3\nline4\nline5")

	// Test tail parameter
	args, _ := json.Marshal(map[string]any{"path": testFile, "tail": 2})
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "read_text_file",
			Arguments: args,
		},
	}

	res, err := s.fsServer.handleReadTextFile(s.ctx, req)
	s.Require().NoError(err)
	s.Require().False(res.IsError)
	s.Equal("line4\nline5", res.Content[0].(*mcp.TextContent).Text)

	// Test tail exceeding file length
	args, _ = json.Marshal(map[string]any{"path": testFile, "tail": 100})
	req.Params.Arguments = args
	res, err = s.fsServer.handleReadTextFile(s.ctx, req)
	s.Require().NoError(err)
	s.Require().False(res.IsError)
	s.Equal("line1\nline2\nline3\nline4\nline5", res.Content[0].(*mcp.TextContent).Text)

	// Test negative tail
	args, _ = json.Marshal(map[string]any{"path": testFile, "tail": -1})
	req.Params.Arguments = args
	res, err = s.fsServer.handleReadTextFile(s.ctx, req)
	s.Require().NoError(err)
	s.Require().True(res.IsError)
	s.Contains(res.Content[0].(*mcp.TextContent).Text, "non-negative")
}

func (s *FilesystemTestSuite) TestReadTextFileHeadAndTailConflict() {
	testFile := s.createFile("lines.txt", "line1\nline2\nline3")

	// Test both head and tail specified (should error)
	args, _ := json.Marshal(map[string]any{"path": testFile, "head": 2, "tail": 2})
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "read_text_file",
			Arguments: args,
		},
	}

	res, err := s.fsServer.handleReadTextFile(s.ctx, req)
	s.Require().NoError(err)
	s.Require().True(res.IsError)
	s.Contains(res.Content[0].(*mcp.TextContent).Text, "Cannot specify both head and tail")
}

func (s *FilesystemTestSuite) TestReadMediaFile() {
	// Setup a test image file
	testFile := s.createFile("image.png", "fake-png-data")

	// Test successful read
	args, _ := json.Marshal(map[string]any{"path": testFile})
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "read_media_file",
			Arguments: args,
		},
	}

	res, err := s.fsServer.handleReadMediaFile(s.ctx, req)
	s.Require().NoError(err)
	s.Require().False(res.IsError)
	s.Require().Len(res.Content, 1)

	imageContent, ok := res.Content[0].(*mcp.ImageContent)
	s.Require().True(ok)
	s.Equal("fake-png-data", string(imageContent.Data))
	s.Equal("image/png", imageContent.MIMEType)

	// Test default MIME type for unknown extension
	testFileUnk := s.createFile("data.unknown", "binary-data")
	args, _ = json.Marshal(map[string]any{"path": testFileUnk})
	req.Params.Arguments = args

	res, err = s.fsServer.handleReadMediaFile(s.ctx, req)
	s.Require().NoError(err)
	s.Require().False(res.IsError)
	imageContent = res.Content[0].(*mcp.ImageContent)
	s.Equal("application/octet-stream", imageContent.MIMEType)

	// Test error cases (invalid path)
	args, _ = json.Marshal(map[string]any{"path": "/etc/passwd"})
	req.Params.Arguments = args
	res, err = s.fsServer.handleReadMediaFile(s.ctx, req)
	s.Require().NoError(err)
	s.Require().True(res.IsError)
}

func (s *FilesystemTestSuite) TestReadMultipleFiles() {
	file1 := s.createFile("multi1.txt", "content 1")
	file2 := s.createFile("multi2.txt", "content 2")
	missingFile := filepath.Join(s.testDir, "missing.txt")
	outsideFile := "/etc/passwd"

	args, _ := json.Marshal(map[string]any{
		"paths": []string{file1, missingFile, outsideFile, file2},
	})
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "read_multiple_files",
			Arguments: args,
		},
	}

	res, err := s.fsServer.handleReadMultipleFiles(s.ctx, req)
	s.Require().NoError(err)
	s.Require().False(res.IsError) // The tool returns partial successes as a valid text response

	text := res.Content[0].(*mcp.TextContent).Text

	// Check successes
	s.Contains(text, "--- "+file1+" ---\ncontent 1")
	s.Contains(text, "--- "+file2+" ---\ncontent 2")

	// Check failures logged in output
	s.Contains(text, missingFile+": Error:")
	s.Contains(text, outsideFile+": Error:")

	// Test invalid JSON arguments
	req.Params.Arguments = []byte(`{"paths": "not-an-array"}`)
	res, err = s.fsServer.handleReadMultipleFiles(s.ctx, req)
	s.Require().NoError(err)
	s.Require().True(res.IsError)
	s.Contains(res.Content[0].(*mcp.TextContent).Text, "Invalid arguments")
}

func (s *FilesystemTestSuite) TestReadTextFileJSONFormat() {
	testFile := s.createFile("hello.txt", "Hello, World!")

	args, _ := json.Marshal(map[string]any{"path": testFile, "format": "json"})
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "read_text_file",
			Arguments: args,
		},
	}

	res, err := s.fsServer.handleReadTextFile(s.ctx, req)
	s.Require().NoError(err)
	s.Require().False(res.IsError)

	var result ReadFileResult
	err = json.Unmarshal([]byte(res.Content[0].(*mcp.TextContent).Text), &result)
	s.Require().NoError(err)
	s.Equal(testFile, result.Path)
	s.Equal("Hello, World!", result.Content)
	s.Equal(int64(13), result.Size)
}

func (s *FilesystemTestSuite) TestReadMultipleFilesJSONFormat() {
	file1 := s.createFile("multi1.txt", "content 1")
	file2 := s.createFile("multi2.txt", "content 2")

	args, _ := json.Marshal(map[string]any{
		"paths":  []string{file1, file2},
		"format": "json",
	})
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "read_multiple_files",
			Arguments: args,
		},
	}

	res, err := s.fsServer.handleReadMultipleFiles(s.ctx, req)
	s.Require().NoError(err)
	s.Require().False(res.IsError)

	var result ReadMultipleFilesResult
	err = json.Unmarshal([]byte(res.Content[0].(*mcp.TextContent).Text), &result)
	s.Require().NoError(err)
	s.Equal(2, result.Summary.Succeeded)
	s.Equal(0, result.Summary.Failed)
	s.Equal(2, len(result.Files))
}
