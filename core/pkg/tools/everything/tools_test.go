package everything

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/suite"
)

type ToolsTestSuite struct {
	suite.Suite
	server *mcp.Server
}

func (s *ToolsTestSuite) SetupTest() {
	s.server = mcp.NewServer(&mcp.Implementation{
		Name:    "mcp-everything",
		Version: "0.1.0",
	}, &mcp.ServerOptions{})
	registerTools(s.server)
}

func TestToolsSuite(t *testing.T) {
	suite.Run(t, new(ToolsTestSuite))
}

func (s *ToolsTestSuite) TestEchoTool() {
	args, _ := json.Marshal(map[string]any{"message": "hello world"})
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "echo",
			Arguments: args,
		},
	}

	res, err := handleEcho(context.Background(), req)
	s.Require().NoError(err)
	s.Require().NotNil(res)
	s.Require().False(res.IsError)
	s.Require().Len(res.Content, 1)

	textContent, ok := res.Content[0].(*mcp.TextContent)
	s.Require().True(ok)
	s.Require().Equal("Echo: hello world", textContent.Text)
}

func (s *ToolsTestSuite) TestGetSumTool() {
	args, _ := json.Marshal(map[string]any{"a": 5, "b": 3})
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "get-sum",
			Arguments: args,
		},
	}

	res, err := handleGetSum(context.Background(), req)
	s.Require().NoError(err)
	s.Require().NotNil(res)
	s.Require().False(res.IsError)

	textContent, ok := res.Content[0].(*mcp.TextContent)
	s.Require().True(ok)
	s.Require().Equal("8", textContent.Text)
}

func (s *ToolsTestSuite) TestGetEnvTool() {
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name: "get-env",
		},
	}

	res, err := handleGetEnv(context.Background(), req)
	s.Require().NoError(err)
	s.Require().NotNil(res)
	s.Require().False(res.IsError)

	textContent, ok := res.Content[0].(*mcp.TextContent)
	s.Require().True(ok)
	s.Require().Contains(textContent.Text, "{")
	s.Require().Contains(textContent.Text, "}")
}

func (s *ToolsTestSuite) TestGetTinyImageTool() {
	req := &mcp.CallToolRequest{}
	res, err := handleGetTinyImage(context.Background(), req)
	s.Require().NoError(err)
	s.Require().NotNil(res)
	s.Require().Len(res.Content, 3)

	_, ok1 := res.Content[0].(*mcp.TextContent)
	s.Require().True(ok1)

	img, ok2 := res.Content[1].(*mcp.ImageContent)
	s.Require().True(ok2)
	s.Require().Equal("image/png", img.MIMEType)
	s.Require().NotEmpty(img.Data)
}

func (s *ToolsTestSuite) TestGetAnnotatedMessageTool() {
	args, _ := json.Marshal(map[string]any{"messageType": "success", "includeImage": false})
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "get-annotated-message",
			Arguments: args,
		},
	}

	res, err := handleGetAnnotatedMessage(context.Background(), req)
	s.Require().NoError(err)
	s.Require().NotNil(res)
	s.Require().Len(res.Content, 1)

	textContent, ok := res.Content[0].(*mcp.TextContent)
	s.Require().True(ok)
	s.Require().Contains(textContent.Text, "successfully")
}

func (s *ToolsTestSuite) TestGetStructuredContentTool() {
	args, _ := json.Marshal(map[string]any{"location": "Chicago"})
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "get-structured-content",
			Arguments: args,
		},
	}

	res, err := handleGetStructuredContent(context.Background(), req)
	s.Require().NoError(err)
	s.Require().NotNil(res)
	s.Require().Len(res.Content, 1)

	textContent, ok := res.Content[0].(*mcp.TextContent)
	s.Require().True(ok)
	s.Require().Contains(textContent.Text, "36")
	s.Require().Contains(textContent.Text, "Light rain / drizzle")
}

func (s *ToolsTestSuite) TestEchoTool_Error() {
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "echo",
			Arguments: json.RawMessage(`invalid`),
		},
	}
	res, err := handleEcho(context.Background(), req)
	s.Require().NoError(err)
	s.Require().NotNil(res)
	s.Require().True(res.IsError)
}

func (s *ToolsTestSuite) TestGetSumTool_Error() {
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "get-sum",
			Arguments: json.RawMessage(`invalid`),
		},
	}
	res, err := handleGetSum(context.Background(), req)
	s.Require().NoError(err)
	s.Require().NotNil(res)
	s.Require().True(res.IsError)
}

func (s *ToolsTestSuite) TestGetAnnotatedMessageTool_Error() {
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "get-annotated-message",
			Arguments: json.RawMessage(`invalid`),
		},
	}
	res, err := handleGetAnnotatedMessage(context.Background(), req)
	s.Require().NoError(err)
	s.Require().NotNil(res)
	s.Require().True(res.IsError)
}

func (s *ToolsTestSuite) TestGetStructuredContentTool_Error() {
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "get-structured-content",
			Arguments: json.RawMessage(`invalid`),
		},
	}
	res, err := handleGetStructuredContent(context.Background(), req)
	s.Require().NoError(err)
	s.Require().NotNil(res)
	s.Require().True(res.IsError)
}

func (s *ToolsTestSuite) TestGetStructuredContentTool_UnknownLocation() {
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "get-structured-content",
			Arguments: json.RawMessage(`{"location":"Atlantis"}`),
		},
	}
	res, err := handleGetStructuredContent(context.Background(), req)
	s.Require().NoError(err)
	s.Require().NotNil(res)
	s.Require().True(res.IsError)

	textContent, ok := res.Content[0].(*mcp.TextContent)
	s.Require().True(ok)
	s.Require().Contains(textContent.Text, "Unknown location: Atlantis")
}

func (s *ToolsTestSuite) TestGetAnnotatedMessageTool_Debug() {
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "get-annotated-message",
			Arguments: json.RawMessage(`{"messageType":"debug", "includeImage":true}`),
		},
	}
	res, err := handleGetAnnotatedMessage(context.Background(), req)
	s.Require().NoError(err)
	s.Require().NotNil(res)
	s.Require().Len(res.Content, 2)

	textContent, ok := res.Content[0].(*mcp.TextContent)
	s.Require().True(ok)
	s.Require().Contains(textContent.Text, "Debug: Cache hit ratio")

	imgContent, ok := res.Content[1].(*mcp.ImageContent)
	s.Require().True(ok)
	s.Require().Equal("image/png", imgContent.MIMEType)
}

func (s *ToolsTestSuite) TestGetAnnotatedMessageTool_ErrorMsg() {
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "get-annotated-message",
			Arguments: json.RawMessage(`{"messageType":"error"}`),
		},
	}
	res, err := handleGetAnnotatedMessage(context.Background(), req)
	s.Require().NoError(err)
	s.Require().NotNil(res)
	s.Require().Len(res.Content, 1)

	textContent, ok := res.Content[0].(*mcp.TextContent)
	s.Require().True(ok)
	s.Require().Contains(textContent.Text, "Error: Operation failed")
}

func (s *ToolsTestSuite) TestGetAnnotatedMessageTool_UnknownMsg() {
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "get-annotated-message",
			Arguments: json.RawMessage(`{"messageType":"foo"}`),
		},
	}
	res, err := handleGetAnnotatedMessage(context.Background(), req)
	s.Require().NoError(err)
	s.Require().NotNil(res)
	s.Require().Len(res.Content, 1)

	textContent, ok := res.Content[0].(*mcp.TextContent)
	s.Require().True(ok)
	s.Require().Contains(textContent.Text, "Unknown message type")
}

func (s *ToolsTestSuite) TestGetStructuredContentTool_NoLocation() {
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "get-structured-content",
			Arguments: json.RawMessage(`{}`),
		},
	}
	res, err := handleGetStructuredContent(context.Background(), req)
	s.Require().NoError(err)
	s.Require().NotNil(res)
	s.Require().True(res.IsError)
}

func (s *ToolsTestSuite) TestGetStructuredContentTool_NewYork() {
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "get-structured-content",
			Arguments: json.RawMessage(`{"location":"New York"}`),
		},
	}
	res, err := handleGetStructuredContent(context.Background(), req)
	s.Require().NoError(err)
	s.Require().NotNil(res)
	s.Require().Len(res.Content, 1)
	
	textContent, ok := res.Content[0].(*mcp.TextContent)
	s.Require().True(ok)
	s.Require().Contains(textContent.Text, "33")
	s.Require().Contains(textContent.Text, "Cloudy")
}

func (s *ToolsTestSuite) TestGetStructuredContentTool_LosAngeles() {
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "get-structured-content",
			Arguments: json.RawMessage(`{"location":"Los Angeles"}`),
		},
	}
	res, err := handleGetStructuredContent(context.Background(), req)
	s.Require().NoError(err)
	s.Require().NotNil(res)
	s.Require().Len(res.Content, 1)
	
	textContent, ok := res.Content[0].(*mcp.TextContent)
	s.Require().True(ok)
	s.Require().Contains(textContent.Text, "73")
	s.Require().Contains(textContent.Text, "Sunny")
}
