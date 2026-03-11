package everything

import (

"testing"
"time"

"github.com/modelcontextprotocol/go-sdk/mcp"
"github.com/stretchr/testify/suite"
)

type SubscriptionsTestSuite struct {
	suite.Suite
}

func TestSubscriptionsSuite(t *testing.T) {
	suite.Run(t, new(SubscriptionsTestSuite))
}

func (s *SubscriptionsTestSuite) TestSubscriptionsToggle() {
	server := mcp.NewServer(&mcp.Implementation{}, &mcp.ServerOptions{})
	s.Require().NotNil(server)
	
	BeginSimulatedResourceUpdates(server, "test-session")
	time.Sleep(10 * time.Millisecond) // Give the goroutine a moment to start and run sendUpdates once
	
	// Re-run should be a no-op
	BeginSimulatedResourceUpdates(server, "test-session")
	
	StopSimulatedResourceUpdates("test-session")
	
	// Re-stop should be safe
	StopSimulatedResourceUpdates("test-session")
	
	// Also test setSubscriptionHandlers
	setSubscriptionHandlers(server)
}

func (s *SubscriptionsTestSuite) TestSendUpdates() {
	server := mcp.NewServer(&mcp.Implementation{}, &mcp.ServerOptions{})
	s.Require().NotNil(server)

	// In test we can't easily catch the internal logger print when sending updates fails, 
// but we can ensure it doesn't panic.
	s.NotPanics(func() {
		sendUpdates(server, "test-session")
	})
}
