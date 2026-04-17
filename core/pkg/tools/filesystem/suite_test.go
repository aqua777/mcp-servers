package filesystem

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/suite"
)

type FilesystemTestSuite struct {
	suite.Suite
	ctx        context.Context
	server     *mcp.Server
	fsServer   *FilesystemServer
	testDir    string
	allowedDir string
}

func (s *FilesystemTestSuite) SetupSuite() {
	s.ctx = context.Background()
	// Use /tmp as the allowed directory base as requested
	s.allowedDir = "/tmp/mcp-fs-test"
	err := os.MkdirAll(s.allowedDir, 0755)
	s.Require().NoError(err)
}

func (s *FilesystemTestSuite) TearDownSuite() {
	err := os.RemoveAll(s.allowedDir)
	s.Require().NoError(err)
}

func (s *FilesystemTestSuite) SetupTest() {
	// Create a unique test directory under the allowed dir for each test
	var err error
	s.testDir, err = os.MkdirTemp(s.allowedDir, "test-*")
	s.Require().NoError(err)

	opts := Options{
		AllowedDirectories: []string{s.allowedDir},
	}
	server, err := NewServer(s.ctx, opts)
	s.Require().NoError(err)
	s.server = server

	// Extract the underlying FilesystemServer to call tools directly if needed,
	// though we usually test via the MCP handlers.
	// We have to recreate fsServer here because NewServer hides it, but we can test the handlers directly.
	s.fsServer = &FilesystemServer{
		allowedDirectories: []string{s.allowedDir},
	}
	s.fsServer.server = mcp.NewServer(&mcp.Implementation{Name: "test", Version: "1"}, nil)
	s.fsServer.registerReadTools()
	s.fsServer.registerWriteTools()
	s.fsServer.registerCopyAppendSymlinkTools()
	s.fsServer.registerListTools()
	s.fsServer.registerEditTools()
	s.fsServer.registerGrepTools()
}

func (s *FilesystemTestSuite) TearDownTest() {
	err := os.RemoveAll(s.testDir)
	s.Require().NoError(err)
}

// Helper to create a file with content
func (s *FilesystemTestSuite) createFile(name string, content string) string {
	path := filepath.Join(s.testDir, name)
	err := os.MkdirAll(filepath.Dir(path), 0755)
	s.Require().NoError(err)
	err = os.WriteFile(path, []byte(content), 0644)
	s.Require().NoError(err)
	return path
}

func TestFilesystemSuite(t *testing.T) {
	suite.Run(t, new(FilesystemTestSuite))
}
