package everything

import (
"context"
"encoding/json"
"testing"

"github.com/modelcontextprotocol/go-sdk/mcp"
"github.com/stretchr/testify/suite"
)

type SamplingTestSuite struct {
	suite.Suite
	server *mcp.Server
}

func (s *SamplingTestSuite) SetupTest() {
	s.server = mcp.NewServer(&mcp.Implementation{
		Name:    "mcp-everything",
		Version: "0.1.0",
	}, &mcp.ServerOptions{})
	
	registerTriggerSamplingRequestTool(s.server)
}

func (s *SamplingTestSuite) TestGetSamplingTool_NoSession() {
	args, _ := json.Marshal(map[string]any{"prompt": "test"})
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name: "trigger-sampling-request",
			Arguments: args,
		},
		Session: nil,
	}
	res, err := handleTriggerSamplingRequest(context.Background(), req)
	s.Require().NoError(err)
	s.Require().NotNil(res)
	
	textContent, ok := res.Content[0].(*mcp.TextContent)
	s.Require().True(ok)
	s.Require().Contains(textContent.Text, "No active session")
}

func TestSamplingSuite(t *testing.T) {
	suite.Run(t, new(SamplingTestSuite))
}
