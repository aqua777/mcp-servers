package everything

import (
	
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/suite"
)

type LoggingTestSuite struct {
	suite.Suite
}

func TestLoggingSuite(t *testing.T) {
	suite.Run(t, new(LoggingTestSuite))
}

func (s *LoggingTestSuite) TestLoggingToggles() {
	server := mcp.NewServer(&mcp.Implementation{}, &mcp.ServerOptions{})
	s.Require().NotNil(server)
	
	BeginSimulatedLogging(server, "test-session")
	time.Sleep(100 * time.Millisecond) // Let it run
	StopSimulatedLogging("test-session")
	
	// Should be safe to call again
	StopSimulatedLogging("test-session")
}
