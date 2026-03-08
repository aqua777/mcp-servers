package filesystem

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"

	"github.com/aqua777/mcp-servers/core/pkg/runtime"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func init() {
	runtime.Register("filesystem", NewServer)
}

type Options struct {
	AllowedDirectories []string
}

type FilesystemServer struct {
	server             *mcp.Server
	allowedDirectories []string
	mu                 sync.RWMutex
}

func (s *FilesystemServer) GetAllowedDirectories() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	dirs := make([]string, len(s.allowedDirectories))
	copy(dirs, s.allowedDirectories)
	return dirs
}

func (s *FilesystemServer) SetAllowedDirectories(dirs []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.allowedDirectories = dirs
}

func (s *FilesystemServer) validatePath(requestedPath string) (string, error) {
	absPath, err := filepath.Abs(requestedPath)
	if err != nil {
		return "", fmt.Errorf("invalid path %s: %v", requestedPath, err)
	}

	allowed := s.GetAllowedDirectories()
	valid, err := IsPathWithinAllowedDirectories(absPath, allowed)
	if err != nil {
		return "", err
	}
	if !valid {
		return "", fmt.Errorf("access denied: path outside allowed directories (%s)", absPath)
	}
	return absPath, nil
}

func NewServer(ctx context.Context, opts any) (*mcp.Server, error) {
	options, ok := opts.(Options)
	if !ok {
		return nil, fmt.Errorf("expected Options, got %T", opts)
	}

	fsServer := &FilesystemServer{
		allowedDirectories: options.AllowedDirectories,
	}

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "mcp-filesystem",
		Version: "0.1.0",
	}, &mcp.ServerOptions{})

	fsServer.server = server

	// Register tools
	fsServer.registerReadTools()
	fsServer.registerWriteTools()
	fsServer.registerListTools()
	fsServer.registerEditTools()

	return server, nil
}
