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
	s.Contains(text, missingFile+": Error reading file")
	s.Contains(text, outsideFile+": Error: access denied")

	// Test invalid JSON arguments
	req.Params.Arguments = []byte(`{"paths": "not-an-array"}`)
	res, err = s.fsServer.handleReadMultipleFiles(s.ctx, req)
	s.Require().NoError(err)
	s.Require().True(res.IsError)
	s.Contains(res.Content[0].(*mcp.TextContent).Text, "Invalid arguments")
}
