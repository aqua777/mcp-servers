package fetch

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/suite"
)

type ServerTestSuite struct {
	suite.Suite
	originalHTTPClientFactory func(proxyURL string) (*http.Client, error)
}

func (suite *ServerTestSuite) SetupSuite() {
	suite.originalHTTPClientFactory = HTTPClientFactory
}

func (suite *ServerTestSuite) TearDownSuite() {
	HTTPClientFactory = suite.originalHTTPClientFactory
}

func (suite *ServerTestSuite) setMockHTTPClient(handler func(req *http.Request) (*http.Response, error)) {
	HTTPClientFactory = func(proxyURL string) (*http.Client, error) {
		return &http.Client{
			Transport: &mockRoundTripper{handler: handler},
		}, nil
	}
}

func (suite *ServerTestSuite) TestNewServer_InvalidOptions() {
	server, err := NewServer(context.Background(), "invalid-options")
	suite.ErrorContains(err, "expected Options")
	suite.Nil(server)
}

func (suite *ServerTestSuite) TestNewServer_ValidOptions() {
	server, err := NewServer(context.Background(), Options{CustomUserAgent: "custom-agent"})
	suite.NoError(err)
	suite.NotNil(server)
}

func (suite *ServerTestSuite) TestFetchToolHandler_ValidRequest() {
	suite.setMockHTTPClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Header:     http.Header{"Content-Type": []string{"text/html"}},
			Body:       io.NopCloser(bytes.NewBufferString(`<html><body><h1>Mocked Tool</h1></body></html>`)),
		}, nil
	})

	handler := fetchToolHandler(Options{IgnoreRobotsTxt: true}, "test-agent")

	args := FetchArgs{URL: "http://example.com", MaxLength: 500}
	argsBytes, _ := json.Marshal(args)

	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Arguments: argsBytes,
		},
	}

	result, err := handler(context.Background(), req)
	suite.NoError(err)
	suite.False(result.IsError)

	textContent, ok := result.Content[0].(*mcp.TextContent)
	suite.True(ok)
	suite.Contains(textContent.Text, "# Mocked Tool")
}

func (suite *ServerTestSuite) TestFetchToolHandler_InvalidJSON() {
	handler := fetchToolHandler(Options{}, "test-agent")

	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Arguments: []byte(`{invalid json`),
		},
	}

	result, err := handler(context.Background(), req)
	suite.NoError(err)
	suite.True(result.IsError)

	textContent, ok := result.Content[0].(*mcp.TextContent)
	suite.True(ok)
	suite.Contains(textContent.Text, "Invalid arguments")
}

func (suite *ServerTestSuite) TestFetchToolHandler_MissingURL() {
	handler := fetchToolHandler(Options{}, "test-agent")

	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Arguments: []byte(`{}`),
		},
	}

	result, err := handler(context.Background(), req)
	suite.NoError(err)
	suite.True(result.IsError)

	textContent, ok := result.Content[0].(*mcp.TextContent)
	suite.True(ok)
	suite.Contains(textContent.Text, "URL is required")
}

func (suite *ServerTestSuite) TestFetchPromptHandler_ValidRequest() {
	suite.setMockHTTPClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Header:     http.Header{"Content-Type": []string{"text/html"}},
			Body:       io.NopCloser(bytes.NewBufferString(`<html><body><h1>Mocked Prompt</h1></body></html>`)),
		}, nil
	})

	handler := fetchPromptHandler(Options{}, "test-agent")

	req := &mcp.GetPromptRequest{
		Params: &mcp.GetPromptParams{
			Arguments: map[string]string{"url": "http://example.com"},
		},
	}

	result, err := handler(context.Background(), req)
	suite.NoError(err)

	suite.Equal("Contents of http://example.com", result.Description)
	suite.Len(result.Messages, 1)

	textContent, ok := result.Messages[0].Content.(*mcp.TextContent)
	suite.True(ok)
	suite.Contains(textContent.Text, "# Mocked Prompt")
}

func (suite *ServerTestSuite) TestFetchPromptHandler_MissingURL() {
	handler := fetchPromptHandler(Options{}, "test-agent")

	req := &mcp.GetPromptRequest{
		Params: &mcp.GetPromptParams{
			Arguments: map[string]string{},
		},
	}

	result, err := handler(context.Background(), req)
	suite.ErrorContains(err, "URL is required")
	suite.Nil(result)
}

func (suite *ServerTestSuite) TestFetchToolHandler_RobotsDisallowed() {
	suite.setMockHTTPClient(func(req *http.Request) (*http.Response, error) {
		// Mock robots.txt response
		if req.URL.Path == "/robots.txt" {
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString("User-agent: *\nDisallow: /")),
			}, nil
		}
		return &http.Response{StatusCode: 404}, nil
	})

	handler := fetchToolHandler(Options{IgnoreRobotsTxt: false}, "test-agent")

	args := FetchArgs{URL: "http://example.com/test"}
	argsBytes, _ := json.Marshal(args)

	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Arguments: argsBytes,
		},
	}

	result, err := handler(context.Background(), req)
	suite.NoError(err)
	suite.True(result.IsError)

	textContent, ok := result.Content[0].(*mcp.TextContent)
	suite.True(ok)
	suite.Contains(textContent.Text, "autonomous fetching of this page is not allowed")
}

func (suite *ServerTestSuite) TestFetchToolHandler_FetchError() {
	suite.setMockHTTPClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 404,
			Body:       io.NopCloser(bytes.NewBufferString("Not Found")),
		}, nil
	})

	handler := fetchToolHandler(Options{IgnoreRobotsTxt: true}, "test-agent")

	args := FetchArgs{URL: "http://example.com/test"}
	argsBytes, _ := json.Marshal(args)

	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Arguments: argsBytes,
		},
	}

	result, err := handler(context.Background(), req)
	suite.NoError(err)
	suite.True(result.IsError)

	textContent, ok := result.Content[0].(*mcp.TextContent)
	suite.True(ok)
	suite.Contains(textContent.Text, "failed to fetch")
}

func (suite *ServerTestSuite) TestFetchPromptHandler_FetchError() {
	suite.setMockHTTPClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 500,
			Body:       io.NopCloser(bytes.NewBufferString("Internal Server Error")),
		}, nil
	})

	handler := fetchPromptHandler(Options{}, "test-agent")

	req := &mcp.GetPromptRequest{
		Params: &mcp.GetPromptParams{
			Arguments: map[string]string{"url": "http://example.com"},
		},
	}

	result, err := handler(context.Background(), req)
	suite.NoError(err)

	suite.Contains(result.Description, "Failed to fetch")
	suite.Len(result.Messages, 1)

	textContent, ok := result.Messages[0].Content.(*mcp.TextContent)
	suite.True(ok)
	suite.Contains(textContent.Text, "status code 500")
}

func TestServerSuite(t *testing.T) {
	suite.Run(t, new(ServerTestSuite))
}
