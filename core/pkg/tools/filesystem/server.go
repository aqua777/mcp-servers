package filesystem

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"

	"github.com/aqua777/mcp-servers/common"
	"github.com/aqua777/mcp-servers/core/pkg/runtime"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func init() {
	runtime.Register(common.MCP_FileSystem, NewServer)
}

type FilesystemServer struct {
	server             *mcp.Server
	allowedDirectories []string
	options            Options
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

// resolveFormat returns the effective output format: per-call override > server default > AIMode > text.
func (s *FilesystemServer) resolveFormat(requestFormat string) string {
	if requestFormat == FormatJSON || requestFormat == FormatText {
		return requestFormat
	}
	if s.options.OutputFormat == FormatJSON || s.options.OutputFormat == FormatText {
		return s.options.OutputFormat
	}
	if s.options.AIMode {
		return FormatJSON
	}
	return FormatText
}

func NewServer(ctx context.Context, opts any) (*mcp.Server, error) {
	options, ok := opts.(Options)
	if !ok {
		return nil, fmt.Errorf("expected Options, got %T", opts)
	}

	fsServer := &FilesystemServer{
		allowedDirectories: options.AllowedDirectories,
		options:            options,
	}

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "mcp-filesystem",
		Version: "0.1.0",
	}, &mcp.ServerOptions{})

	fsServer.server = server

	// Register tools
	fsServer.registerReadTools()
	fsServer.registerWriteTools()
	fsServer.registerCopyAppendSymlinkTools()
	fsServer.registerListTools()
	fsServer.registerEditTools()
	fsServer.registerGrepTools()

	return server, nil
}
