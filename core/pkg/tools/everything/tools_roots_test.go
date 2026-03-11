package everything

import (
"context"
"testing"

"github.com/modelcontextprotocol/go-sdk/mcp"
"github.com/stretchr/testify/suite"
)

type RootsTestSuite struct {
	suite.Suite
	server *mcp.Server
}

func (s *RootsTestSuite) SetupTest() {
	s.server = mcp.NewServer(&mcp.Implementation{
		Name:    "mcp-everything",
		Version: "0.1.0",
	}, &mcp.ServerOptions{})
	
	registerGetRootsListTool(s.server)
}

func (s *RootsTestSuite) TestGetRootsListTool_NoSession() {
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name: "get-roots-list",
		},
		Session: nil,
	}
	res, err := handleGetRootsList(context.Background(), req)
	s.Require().NoError(err)
	s.Require().NotNil(res)
	
	textContent, ok := res.Content[0].(*mcp.TextContent)
	s.Require().True(ok)
	s.Require().Contains(textContent.Text, "No active session")
}

func TestRootsSuite(t *testing.T) {
	suite.Run(t, new(RootsTestSuite))
}
