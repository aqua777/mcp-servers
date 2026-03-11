package everything

import (
"context"
"testing"

"github.com/modelcontextprotocol/go-sdk/mcp"
"github.com/stretchr/testify/suite"
)

type ElicitationTestSuite struct {
	suite.Suite
	server *mcp.Server
}

func (s *ElicitationTestSuite) SetupTest() {
	s.server = mcp.NewServer(&mcp.Implementation{
		Name:    "mcp-everything",
		Version: "0.1.0",
	}, &mcp.ServerOptions{})
	
	registerTriggerElicitationRequestTool(s.server)
}

func (s *ElicitationTestSuite) TestGetElicitationTool_NoSession() {
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name: "trigger-elicitation-request",
		},
		Session: nil,
	}
	res, err := handleTriggerElicitationRequest(context.Background(), req)
	s.Require().NoError(err)
	s.Require().NotNil(res)
	
	textContent, ok := res.Content[0].(*mcp.TextContent)
	s.Require().True(ok)
	s.Require().Contains(textContent.Text, "No active session")
}

func TestElicitationSuite(t *testing.T) {
	suite.Run(t, new(ElicitationTestSuite))
}
