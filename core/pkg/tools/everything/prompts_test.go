package everything

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/suite"
)

type PromptsTestSuite struct {
	suite.Suite
	server *mcp.Server
}

func (s *PromptsTestSuite) SetupTest() {
	s.server = mcp.NewServer(&mcp.Implementation{
		Name:    "mcp-everything",
		Version: "0.1.0",
	}, &mcp.ServerOptions{})
	registerPrompts(s.server)
}

func TestPromptsSuite(t *testing.T) {
	suite.Run(t, new(PromptsTestSuite))
}

func (s *PromptsTestSuite) TestSimplePrompt() {
	req := &mcp.GetPromptRequest{
		Params: &mcp.GetPromptParams{
			Name: "simple-prompt",
		},
	}

	res, err := handleSimplePrompt(context.Background(), req)
	s.Require().NoError(err)
	s.Require().NotNil(res)
	s.Require().Equal("Simple Prompt", res.Description)
	s.Require().Len(res.Messages, 1)
	
	msg := res.Messages[0]
	s.Require().Equal(mcp.Role("user"), msg.Role)
	
	textContent, ok := msg.Content.(*mcp.TextContent)
	s.Require().True(ok)
	s.Require().Contains(textContent.Text, "simple prompt without arguments")
}

func (s *PromptsTestSuite) TestArgsPrompt() {
	req := &mcp.GetPromptRequest{
		Params: &mcp.GetPromptParams{
			Name: "args-prompt",
			Arguments: map[string]string{
				"city":  "San Francisco",
				"state": "CA",
			},
		},
	}

	res, err := handleArgsPrompt(context.Background(), req)
	s.Require().NoError(err)
	s.Require().NotNil(res)
	s.Require().Equal("Args Prompt", res.Description)
	s.Require().Len(res.Messages, 1)

	textContent, ok := res.Messages[0].Content.(*mcp.TextContent)
	s.Require().True(ok)
	s.Require().Equal("Tell me about the weather in San Francisco, CA.", textContent.Text)
}

func (s *PromptsTestSuite) TestArgsPrompt_MissingCity() {
	req := &mcp.GetPromptRequest{
		Params: &mcp.GetPromptParams{
			Name: "args-prompt",
			Arguments: map[string]string{
				"state": "CA",
			},
		},
	}

	res, err := handleArgsPrompt(context.Background(), req)
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "city argument is required")
	s.Require().Nil(res)
}

func (s *PromptsTestSuite) TestArgsPrompt_MissingState() {
	req := &mcp.GetPromptRequest{
		Params: &mcp.GetPromptParams{
			Name: "args-prompt",
			Arguments: map[string]string{
				"city": "London",
			},
		},
	}

	res, err := handleArgsPrompt(context.Background(), req)
	s.Require().NoError(err)
	s.Require().NotNil(res)
	
	textContent, ok := res.Messages[0].Content.(*mcp.TextContent)
	s.Require().True(ok)
	s.Require().Equal("Tell me about the weather in London.", textContent.Text)
}
