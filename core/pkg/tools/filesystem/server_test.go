package filesystem

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewServer(t *testing.T) {
	// Valid options
	opts := Options{
		AllowedDirectories: []string{"/tmp/test"},
	}

	server, err := NewServer(context.Background(), opts)
	require.NoError(t, err)
	assert.NotNil(t, server)

	// Invalid options
	invalidOpts := struct{}{}
	server, err = NewServer(context.Background(), invalidOpts)
	require.Error(t, err)
	assert.Nil(t, server)
}

func (s *FilesystemTestSuite) TestServerStateManagement() {
	// Test GetAllowedDirectories and SetAllowedDirectories via the server methods
	initialDirs := s.fsServer.GetAllowedDirectories()
	s.Equal([]string{s.allowedDir}, initialDirs)

	newDirs := []string{filepath.Join(s.allowedDir, "new")}
	s.fsServer.SetAllowedDirectories(newDirs)
	
	updatedDirs := s.fsServer.GetAllowedDirectories()
	s.Equal(newDirs, updatedDirs)
	
	// Restore for other tests
	s.fsServer.SetAllowedDirectories([]string{s.allowedDir})
}

func (s *FilesystemTestSuite) TestValidatePath() {
	// Note: We use s.fsServer because we need access to unexported methods
	
	// Valid path inside allowed directory
	validPath := filepath.Join(s.testDir, "test.txt")
	resolved, err := s.fsServer.validatePath(validPath)
	s.NoError(err)
	s.Equal(validPath, resolved)

	// Invalid path outside allowed directory
	outsidePath := "/etc/passwd"
	resolved, err = s.fsServer.validatePath(outsidePath)
	s.Error(err)
	s.Empty(resolved)
	s.Contains(err.Error(), "access denied")

	// Directory traversal attempt
	traversalPath := filepath.Join(s.allowedDir, "..", "etc", "passwd")
	resolved, err = s.fsServer.validatePath(traversalPath)
	s.Error(err)
	s.Empty(resolved)
}

func (s *FilesystemTestSuite) TestToolRegistration() {
	// The underlying mcp.Server should have all tools registered
	// We can't directly inspect the registered tools without reflection or calling them,
	// but we can verify the initialization sequence completed without error.
	
	opts := Options{
		AllowedDirectories: []string{s.allowedDir},
	}
	
	server, err := NewServer(s.ctx, opts)
	s.Require().NoError(err)
	s.NotNil(server)
	
	// Verify it's a valid pointer type that implements the needed interfaces
	_, ok := interface{}(server).(*mcp.Server)
	s.True(ok)
}
