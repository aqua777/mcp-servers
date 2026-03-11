package runtime

import (
    "context"
    "errors"
    "testing"

    "github.com/modelcontextprotocol/go-sdk/mcp"
    "github.com/stretchr/testify/suite"
)

type RuntimeTestSuite struct {
    suite.Suite

    originalRegistry        map[string]ServerFactory
    originalTransportFactory func() mcp.Transport
    originalServerRunner     func(*mcp.Server, context.Context, mcp.Transport) error
}

func TestRuntimeTestSuite(t *testing.T) {
    suite.Run(t, new(RuntimeTestSuite))
}

func (s *RuntimeTestSuite) SetupTest() {
    registryMu.Lock()
    s.originalRegistry = make(map[string]ServerFactory, len(registry))
    for name, factory := range registry {
        s.originalRegistry[name] = factory
    }
    registry = map[string]ServerFactory{}
    registryMu.Unlock()

    s.originalTransportFactory = transportFactory
    s.originalServerRunner = serverRunner
}

func (s *RuntimeTestSuite) TearDownTest() {
    registryMu.Lock()
    registry = s.originalRegistry
    registryMu.Unlock()

    transportFactory = s.originalTransportFactory
    serverRunner = s.originalServerRunner
}

func (s *RuntimeTestSuite) TestRegisterPanicsOnDuplicate() {
    Register("duplicate", func(context.Context, any) (*mcp.Server, error) {
        return nil, nil
    })

    s.Panics(func() {
        Register("duplicate", nil)
    })
}

func (s *RuntimeTestSuite) TestRunReturnsErrorForUnknownServer() {
    err := Run(context.Background(), "missing", nil)
    s.Error(err)
    s.Contains(err.Error(), "unknown server \"missing\"")
}

func (s *RuntimeTestSuite) TestRunReturnsErrorForNilFactory() {
    registryMu.Lock()
    registry["nilFactory"] = nil
    registryMu.Unlock()

    err := Run(context.Background(), "nilFactory", nil)
    s.EqualError(err, "runtime: nil factory registered")
}

func (s *RuntimeTestSuite) TestRunPropagatesFactoryErrors() {
    expectedErr := errors.New("boom")
    Register("factoryError", func(context.Context, any) (*mcp.Server, error) {
        return nil, expectedErr
    })

    err := Run(context.Background(), "factoryError", nil)
    s.Error(err)
    s.Contains(err.Error(), "runtime: building factoryError server")
    s.ErrorIs(err, expectedErr)
}

func (s *RuntimeTestSuite) TestRunRejectsNilServer() {
    Register("nilServer", func(context.Context, any) (*mcp.Server, error) {
        return nil, nil
    })

    err := Run(context.Background(), "nilServer", nil)
    s.EqualError(err, "runtime: factory returned nil server")
}

func (s *RuntimeTestSuite) TestRunSuccessInvokesServerRunner() {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    opts := map[string]string{"key": "value"}
    server := &mcp.Server{}
    transport := &fakeTransport{}

    Register("success", func(receivedCtx context.Context, receivedOpts any) (*mcp.Server, error) {
        s.Equal(ctx, receivedCtx)
        s.Equal(opts, receivedOpts)
        return server, nil
    })

    transportFactory = func() mcp.Transport {
        return transport
    }

    called := false
    serverRunner = func(srv *mcp.Server, runCtx context.Context, t mcp.Transport) error {
        called = true
        s.Equal(server, srv)
        s.Equal(ctx, runCtx)
        s.Equal(transport, t)
        return nil
    }

    err := Run(ctx, "success", opts)
    s.NoError(err)
    s.True(called)
}

type fakeTransport struct{}

func (f *fakeTransport) Connect(context.Context) (mcp.Connection, error) {
    return nil, errors.New("not implemented")
}
