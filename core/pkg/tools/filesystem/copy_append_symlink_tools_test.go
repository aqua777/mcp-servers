package filesystem

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ---- copy_file tests ----

func (s *FilesystemTestSuite) TestCopyFileSingleFile() {
	src := s.createFile("src.txt", "hello copy")
	dst := filepath.Join(s.testDir, "dst.txt")

	args, _ := json.Marshal(map[string]any{
		"source":      src,
		"destination": dst,
	})
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{Name: ToolCopyFile, Arguments: args},
	}

	res, err := s.fsServer.handleCopyFile(s.ctx, req)
	s.Require().NoError(err)
	s.Require().False(res.IsError)
	s.Contains(res.Content[0].(*mcp.TextContent).Text, "Successfully copied")

	content, err := os.ReadFile(dst)
	s.Require().NoError(err)
	s.Equal("hello copy", string(content))

	// Source should still exist
	_, err = os.Stat(src)
	s.Require().NoError(err)
}

func (s *FilesystemTestSuite) TestCopyFileDestinationAlreadyExists() {
	src := s.createFile("src.txt", "hello")
	dst := s.createFile("dst.txt", "already here")

	args, _ := json.Marshal(map[string]any{
		"source":      src,
		"destination": dst,
	})
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{Name: ToolCopyFile, Arguments: args},
	}

	res, err := s.fsServer.handleCopyFile(s.ctx, req)
	s.Require().NoError(err)
	s.Require().True(res.IsError)
	s.Contains(res.Content[0].(*mcp.TextContent).Text, "already exists")
}

func (s *FilesystemTestSuite) TestCopyFileDirectoryWithoutRecursive() {
	srcDir := filepath.Join(s.testDir, "srcdir")
	s.Require().NoError(os.MkdirAll(srcDir, 0755))
	s.createFile("srcdir/a.txt", "a")

	dst := filepath.Join(s.testDir, "dstdir")

	args, _ := json.Marshal(map[string]any{
		"source":      srcDir,
		"destination": dst,
	})
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{Name: ToolCopyFile, Arguments: args},
	}

	res, err := s.fsServer.handleCopyFile(s.ctx, req)
	s.Require().NoError(err)
	s.Require().True(res.IsError)
	s.Contains(res.Content[0].(*mcp.TextContent).Text, "recursive=true")
}

func (s *FilesystemTestSuite) TestCopyFileDirectoryRecursive() {
	srcDir := filepath.Join(s.testDir, "srcdir")
	s.Require().NoError(os.MkdirAll(filepath.Join(srcDir, "sub"), 0755))
	s.createFile("srcdir/a.txt", "aaa")
	s.createFile("srcdir/sub/b.txt", "bbb")

	dst := filepath.Join(s.testDir, "dstdir")

	args, _ := json.Marshal(map[string]any{
		"source":      srcDir,
		"destination": dst,
		"recursive":   true,
	})
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{Name: ToolCopyFile, Arguments: args},
	}

	res, err := s.fsServer.handleCopyFile(s.ctx, req)
	s.Require().NoError(err)
	s.Require().False(res.IsError)

	aContent, err := os.ReadFile(filepath.Join(dst, "a.txt"))
	s.Require().NoError(err)
	s.Equal("aaa", string(aContent))

	bContent, err := os.ReadFile(filepath.Join(dst, "sub", "b.txt"))
	s.Require().NoError(err)
	s.Equal("bbb", string(bContent))
}

func (s *FilesystemTestSuite) TestCopyFileGlobPattern() {
	s.createFile("g1.go", "package g1")
	s.createFile("g2.go", "package g2")
	s.createFile("other.txt", "not go")

	dstDir := filepath.Join(s.testDir, "godst")

	globPattern := filepath.Join(s.testDir, "*.go")
	args, _ := json.Marshal(map[string]any{
		"source":      globPattern,
		"destination": dstDir,
	})
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{Name: ToolCopyFile, Arguments: args},
	}

	res, err := s.fsServer.handleCopyFile(s.ctx, req)
	s.Require().NoError(err)
	s.Require().False(res.IsError)

	_, err = os.Stat(filepath.Join(dstDir, "g1.go"))
	s.Require().NoError(err)
	_, err = os.Stat(filepath.Join(dstDir, "g2.go"))
	s.Require().NoError(err)
	_, err = os.Stat(filepath.Join(dstDir, "other.txt"))
	s.Require().Error(err) // should not have been copied
}

func (s *FilesystemTestSuite) TestCopyFileGlobNoMatches() {
	globPattern := filepath.Join(s.testDir, "*.nonexistent")
	dst := filepath.Join(s.testDir, "dst")

	args, _ := json.Marshal(map[string]any{
		"source":      globPattern,
		"destination": dst,
	})
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{Name: ToolCopyFile, Arguments: args},
	}

	res, err := s.fsServer.handleCopyFile(s.ctx, req)
	s.Require().NoError(err)
	s.Require().True(res.IsError)
	s.Contains(res.Content[0].(*mcp.TextContent).Text, "No files match")
}

func (s *FilesystemTestSuite) TestCopyFileGlobDirectoryWithoutRecursive() {
	srcDir := filepath.Join(s.testDir, "adir")
	s.Require().NoError(os.MkdirAll(srcDir, 0755))

	globPattern := filepath.Join(s.testDir, "adir")
	dst := filepath.Join(s.testDir, "dst")

	args, _ := json.Marshal(map[string]any{
		"source":      globPattern,
		"destination": dst,
	})
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{Name: ToolCopyFile, Arguments: args},
	}

	// adir is not a glob, so it falls into plain path mode — but we need a glob to test glob+dir branch.
	// Use a glob that matches a directory:
	s.createFile("adir/file.txt", "inside")
	globPattern2 := filepath.Join(s.testDir, "ad*")
	args2, _ := json.Marshal(map[string]any{
		"source":      globPattern2,
		"destination": dst,
	})
	req.Params.Arguments = args2

	res, err := s.fsServer.handleCopyFile(s.ctx, req)
	s.Require().NoError(err)
	s.Require().True(res.IsError)
	s.Contains(res.Content[0].(*mcp.TextContent).Text, "recursive=true")
}

func (s *FilesystemTestSuite) TestCopyFileGlobWithExcludePatterns() {
	s.createFile("p1.txt", "keep")
	s.createFile("p2.skip", "skip")

	dstDir := filepath.Join(s.testDir, "pdst")
	globPattern := filepath.Join(s.testDir, "p*")

	args, _ := json.Marshal(map[string]any{
		"source":          globPattern,
		"destination":     dstDir,
		"excludePatterns": []string{"*.skip"},
	})
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{Name: ToolCopyFile, Arguments: args},
	}

	res, err := s.fsServer.handleCopyFile(s.ctx, req)
	s.Require().NoError(err)
	// p2.skip is a file, not a directory, so excludePatterns don't apply at the glob-match level
	// (excludePatterns only filter recursive dir traversal). Both files will be copied.
	// This test verifies the tool succeeds and copies at least p1.txt.
	s.Require().False(res.IsError)
	_, err = os.Stat(filepath.Join(dstDir, "p1.txt"))
	s.Require().NoError(err)
}

func (s *FilesystemTestSuite) TestCopyFileRecursiveWithExcludePatterns() {
	srcDir := filepath.Join(s.testDir, "excl_src")
	s.Require().NoError(os.MkdirAll(srcDir, 0755))
	s.createFile("excl_src/keep.go", "keep")
	s.createFile("excl_src/skip.txt", "skip")

	dst := filepath.Join(s.testDir, "excl_dst")

	args, _ := json.Marshal(map[string]any{
		"source":          srcDir,
		"destination":     dst,
		"recursive":       true,
		"excludePatterns": []string{"*.txt"},
	})
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{Name: ToolCopyFile, Arguments: args},
	}

	res, err := s.fsServer.handleCopyFile(s.ctx, req)
	s.Require().NoError(err)
	s.Require().False(res.IsError)

	_, err = os.Stat(filepath.Join(dst, "keep.go"))
	s.Require().NoError(err)
	_, err = os.Stat(filepath.Join(dst, "skip.txt"))
	s.Require().True(os.IsNotExist(err))
}

func (s *FilesystemTestSuite) TestCopyFileSourceOutsideAllowed() {
	args, _ := json.Marshal(map[string]any{
		"source":      "/etc/passwd",
		"destination": filepath.Join(s.testDir, "dst.txt"),
	})
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{Name: ToolCopyFile, Arguments: args},
	}

	res, err := s.fsServer.handleCopyFile(s.ctx, req)
	s.Require().NoError(err)
	s.Require().True(res.IsError)
}

func (s *FilesystemTestSuite) TestCopyFileDestinationOutsideAllowed() {
	src := s.createFile("src.txt", "hello")

	args, _ := json.Marshal(map[string]any{
		"source":      src,
		"destination": "/etc/dst.txt",
	})
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{Name: ToolCopyFile, Arguments: args},
	}

	res, err := s.fsServer.handleCopyFile(s.ctx, req)
	s.Require().NoError(err)
	s.Require().True(res.IsError)
}

func (s *FilesystemTestSuite) TestCopyFileInvalidJSON() {
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{Name: ToolCopyFile, Arguments: []byte(`{"source": 123}`)},
	}

	res, err := s.fsServer.handleCopyFile(s.ctx, req)
	s.Require().NoError(err)
	s.Require().True(res.IsError)
}

func (s *FilesystemTestSuite) TestCopyFileSourceNotExist() {
	args, _ := json.Marshal(map[string]any{
		"source":      filepath.Join(s.testDir, "nope.txt"),
		"destination": filepath.Join(s.testDir, "dst.txt"),
	})
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{Name: ToolCopyFile, Arguments: args},
	}

	res, err := s.fsServer.handleCopyFile(s.ctx, req)
	s.Require().NoError(err)
	s.Require().True(res.IsError)
}

func (s *FilesystemTestSuite) TestCopyFileJSONFormat() {
	src := s.createFile("jsrc.txt", "json copy")
	dst := filepath.Join(s.testDir, "jdst.txt")

	args, _ := json.Marshal(map[string]any{
		"source":      src,
		"destination": dst,
		"format":      "json",
	})
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{Name: ToolCopyFile, Arguments: args},
	}

	res, err := s.fsServer.handleCopyFile(s.ctx, req)
	s.Require().NoError(err)
	s.Require().False(res.IsError)

	var result CopyResult
	s.Require().NoError(json.Unmarshal([]byte(res.Content[0].(*mcp.TextContent).Text), &result))
	s.Equal("ok", result.Status)
	s.Equal(1, result.Summary.FilesCopied)
	s.EqualValues(9, result.Summary.BytesCopied)
}

// ---- append_file tests ----

func (s *FilesystemTestSuite) TestAppendFileToExisting() {
	f := s.createFile("append.txt", "first line\n")

	args, _ := json.Marshal(map[string]any{
		"path":    f,
		"content": "second line\n",
	})
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{Name: ToolAppendFile, Arguments: args},
	}

	res, err := s.fsServer.handleAppendFile(s.ctx, req)
	s.Require().NoError(err)
	s.Require().False(res.IsError)
	s.Contains(res.Content[0].(*mcp.TextContent).Text, "appended")

	content, err := os.ReadFile(f)
	s.Require().NoError(err)
	s.Equal("first line\nsecond line\n", string(content))
}

func (s *FilesystemTestSuite) TestAppendFileCreatesNew() {
	newFile := filepath.Join(s.testDir, "new_append.txt")

	args, _ := json.Marshal(map[string]any{
		"path":    newFile,
		"content": "created via append\n",
	})
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{Name: ToolAppendFile, Arguments: args},
	}

	res, err := s.fsServer.handleAppendFile(s.ctx, req)
	s.Require().NoError(err)
	s.Require().False(res.IsError)
	s.Contains(res.Content[0].(*mcp.TextContent).Text, "created")

	content, err := os.ReadFile(newFile)
	s.Require().NoError(err)
	s.Equal("created via append\n", string(content))
}

func (s *FilesystemTestSuite) TestAppendFileOutsideAllowed() {
	args, _ := json.Marshal(map[string]any{
		"path":    "/etc/passwd",
		"content": "hacked",
	})
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{Name: ToolAppendFile, Arguments: args},
	}

	res, err := s.fsServer.handleAppendFile(s.ctx, req)
	s.Require().NoError(err)
	s.Require().True(res.IsError)
}

func (s *FilesystemTestSuite) TestAppendFileInvalidJSON() {
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{Name: ToolAppendFile, Arguments: []byte(`{"path": 123}`)},
	}

	res, err := s.fsServer.handleAppendFile(s.ctx, req)
	s.Require().NoError(err)
	s.Require().True(res.IsError)
}

func (s *FilesystemTestSuite) TestAppendFileJSONFormat() {
	f := s.createFile("ajson.txt", "base\n")

	args, _ := json.Marshal(map[string]any{
		"path":    f,
		"content": "appended\n",
		"format":  "json",
	})
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{Name: ToolAppendFile, Arguments: args},
	}

	res, err := s.fsServer.handleAppendFile(s.ctx, req)
	s.Require().NoError(err)
	s.Require().False(res.IsError)

	var result AppendResult
	s.Require().NoError(json.Unmarshal([]byte(res.Content[0].(*mcp.TextContent).Text), &result))
	s.Equal("ok", result.Status)
	s.Equal(9, result.BytesWritten)
	s.False(result.Created)
}

func (s *FilesystemTestSuite) TestAppendFileJSONFormatCreated() {
	newFile := filepath.Join(s.testDir, "ajson_new.txt")

	args, _ := json.Marshal(map[string]any{
		"path":    newFile,
		"content": "hello",
		"format":  "json",
	})
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{Name: ToolAppendFile, Arguments: args},
	}

	res, err := s.fsServer.handleAppendFile(s.ctx, req)
	s.Require().NoError(err)
	s.Require().False(res.IsError)

	var result AppendResult
	s.Require().NoError(json.Unmarshal([]byte(res.Content[0].(*mcp.TextContent).Text), &result))
	s.True(result.Created)
}

// ---- create_symlink tests ----

func (s *FilesystemTestSuite) TestCreateSymlinkToFile() {
	target := s.createFile("real.txt", "real content")
	linkPath := filepath.Join(s.testDir, "link.txt")

	args, _ := json.Marshal(map[string]any{
		"target": target,
		"path":   linkPath,
	})
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{Name: ToolCreateSymlink, Arguments: args},
	}

	res, err := s.fsServer.handleCreateSymlink(s.ctx, req)
	s.Require().NoError(err)
	s.Require().False(res.IsError)
	s.Contains(res.Content[0].(*mcp.TextContent).Text, "Successfully created symlink")

	// Verify symlink resolves correctly
	content, err := os.ReadFile(linkPath)
	s.Require().NoError(err)
	s.Equal("real content", string(content))
}

func (s *FilesystemTestSuite) TestCreateSymlinkToDirectory() {
	srcDir := filepath.Join(s.testDir, "realdir")
	s.Require().NoError(os.MkdirAll(srcDir, 0755))
	s.createFile("realdir/file.txt", "inside dir")

	linkPath := filepath.Join(s.testDir, "dirlink")

	args, _ := json.Marshal(map[string]any{
		"target": srcDir,
		"path":   linkPath,
	})
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{Name: ToolCreateSymlink, Arguments: args},
	}

	res, err := s.fsServer.handleCreateSymlink(s.ctx, req)
	s.Require().NoError(err)
	s.Require().False(res.IsError)

	// Symlink to a directory should resolve files inside it
	content, err := os.ReadFile(filepath.Join(linkPath, "file.txt"))
	s.Require().NoError(err)
	s.Equal("inside dir", string(content))
}

func (s *FilesystemTestSuite) TestCreateSymlinkAlreadyExists() {
	target := s.createFile("real2.txt", "real")
	existing := s.createFile("existing_link.txt", "existing")

	args, _ := json.Marshal(map[string]any{
		"target": target,
		"path":   existing,
	})
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{Name: ToolCreateSymlink, Arguments: args},
	}

	res, err := s.fsServer.handleCreateSymlink(s.ctx, req)
	s.Require().NoError(err)
	s.Require().True(res.IsError)
	s.Contains(res.Content[0].(*mcp.TextContent).Text, "already exists")
}

func (s *FilesystemTestSuite) TestCreateSymlinkTargetOutsideAllowed() {
	linkPath := filepath.Join(s.testDir, "link.txt")

	args, _ := json.Marshal(map[string]any{
		"target": "/etc/passwd",
		"path":   linkPath,
	})
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{Name: ToolCreateSymlink, Arguments: args},
	}

	res, err := s.fsServer.handleCreateSymlink(s.ctx, req)
	s.Require().NoError(err)
	s.Require().True(res.IsError)
}

func (s *FilesystemTestSuite) TestCreateSymlinkPathOutsideAllowed() {
	target := s.createFile("real3.txt", "real")

	args, _ := json.Marshal(map[string]any{
		"target": target,
		"path":   "/etc/my_link",
	})
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{Name: ToolCreateSymlink, Arguments: args},
	}

	res, err := s.fsServer.handleCreateSymlink(s.ctx, req)
	s.Require().NoError(err)
	s.Require().True(res.IsError)
}

func (s *FilesystemTestSuite) TestCreateSymlinkInvalidJSON() {
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{Name: ToolCreateSymlink, Arguments: []byte(`{"path": 123}`)},
	}

	res, err := s.fsServer.handleCreateSymlink(s.ctx, req)
	s.Require().NoError(err)
	s.Require().True(res.IsError)
}

func (s *FilesystemTestSuite) TestCreateSymlinkJSONFormat() {
	target := s.createFile("real_json.txt", "data")
	linkPath := filepath.Join(s.testDir, "json_link.txt")

	args, _ := json.Marshal(map[string]any{
		"target": target,
		"path":   linkPath,
		"format": "json",
	})
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{Name: ToolCreateSymlink, Arguments: args},
	}

	res, err := s.fsServer.handleCreateSymlink(s.ctx, req)
	s.Require().NoError(err)
	s.Require().False(res.IsError)

	var result SymlinkResult
	s.Require().NoError(json.Unmarshal([]byte(res.Content[0].(*mcp.TextContent).Text), &result))
	s.Equal("ok", result.Status)
	s.Equal(target, result.Target)
	s.Equal(linkPath, result.Path)
}
