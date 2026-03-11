package everything

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"
)

type ServerTestSuite struct {
	suite.Suite
}

func TestServerSuite(t *testing.T) {
	suite.Run(t, new(ServerTestSuite))
}

func (s *ServerTestSuite) TestNewServer() {
	server, err := NewServer(context.Background(), nil)
	s.Require().NoError(err)
	s.Require().NotNil(server)
}

func (s *ServerTestSuite) TestLoggingToggles() {
	server, err := NewServer(context.Background(), nil)
	s.Require().NoError(err)
	s.Require().NotNil(server)
	
	// Should not panic or block indefinitely
	BeginSimulatedLogging(server, "test-session")
	BeginSimulatedLogging(server, "test-session") // call twice to check deduplication logic
	StopSimulatedLogging("test-session")
	StopSimulatedLogging("test-session") // call twice to check idempotent stop
}

func (s *ServerTestSuite) TestSubscriptionToggles() {
	server, err := NewServer(context.Background(), nil)
	s.Require().NoError(err)
	s.Require().NotNil(server)
	
	BeginSimulatedResourceUpdates(server, "test-session")
	BeginSimulatedResourceUpdates(server, "test-session") 
	StopSimulatedResourceUpdates("test-session")
	StopSimulatedResourceUpdates("test-session") 
}
