package runtime

import (
    "context"
    "errors"
    "fmt"
    "sync"

    "github.com/modelcontextprotocol/go-sdk/mcp"
)

// ServerFactory constructs an MCP server using the provided options.
//
// opts is server-specific; callers must provide the exact struct required by the
// registered server factory.
type ServerFactory func(ctx context.Context, opts any) (*mcp.Server, error)

var (
    registryMu sync.RWMutex
    registry   = map[string]ServerFactory{}
)

// Register wires a server factory under the supplied name. Panics if the name
// is already taken.
func Register(name string, factory ServerFactory) {
    registryMu.Lock()
    defer registryMu.Unlock()
    if _, exists := registry[name]; exists {
        panic(fmt.Sprintf("runtime: server %q already registered", name))
    }
    registry[name] = factory
}

// Run builds the named server with opts and executes it using the stdio
// transport.
func Run(ctx context.Context, name string, opts any) error {
    registryMu.RLock()
    factory, ok := registry[name]
    registryMu.RUnlock()
    if !ok {
        return fmt.Errorf("runtime: unknown server %q", name)
    }
    if factory == nil {
        return errors.New("runtime: nil factory registered")
    }
    server, err := factory(ctx, opts)
    if err != nil {
        return fmt.Errorf("runtime: building %s server: %w", name, err)
    }
    if server == nil {
        return errors.New("runtime: factory returned nil server")
    }
    return server.Run(ctx, &mcp.StdioTransport{})
}
