package fetch

import (
	"fmt"
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/suite"
)

type ContentTestSuite struct {
	suite.Suite
	originalHTTPClientFactory func(proxyURL string) (*http.Client, error)
}

func (suite *ContentTestSuite) SetupSuite() {
	suite.originalHTTPClientFactory = HTTPClientFactory
}

func (suite *ContentTestSuite) TearDownSuite() {
	HTTPClientFactory = suite.originalHTTPClientFactory
}

// mockRoundTripper intercepts HTTP requests and returns a mocked response.
type mockRoundTripper struct {
	handler func(req *http.Request) (*http.Response, error)
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.handler(req)
}

func (suite *ContentTestSuite) setMockHTTPClient(handler func(req *http.Request) (*http.Response, error)) {
	HTTPClientFactory = func(proxyURL string) (*http.Client, error) {
		return &http.Client{
			Transport: &mockRoundTripper{handler: handler},
		}, nil
	}
}

func (suite *ContentTestSuite) TestExtractContentFromHTML() {
	html := `<html><head><title>Test</title></head><body><h1>Hello World</h1><p>This is a test.</p></body></html>`
	content, err := extractContentFromHTML(html)
	suite.NoError(err)
	suite.Contains(content, "# Hello World")
	suite.Contains(content, "This is a test.")
}

func (suite *ContentTestSuite) TestExtractContentFromHTMLEmpty() {
	html := `<html></html>`
	content, err := extractContentFromHTML(html)
	suite.NoError(err)
	suite.Equal("<error>Page failed to be simplified from HTML</error>", content)
}

func (suite *ContentTestSuite) TestFetchURL_SuccessHTML() {
	suite.setMockHTTPClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Header:     http.Header{"Content-Type": []string{"text/html"}},
			Body:       io.NopCloser(bytes.NewBufferString(`<html><body><h1>Mocked Content</h1></body></html>`)),
		}, nil
	})

	content, prefix, err := fetchURL(context.Background(), "http://example.com", "test-agent", false, "")
	suite.NoError(err)
	suite.Empty(prefix)
	suite.Contains(content, "# Mocked Content")
}

func (suite *ContentTestSuite) TestFetchURL_Raw() {
	suite.setMockHTTPClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Header:     http.Header{"Content-Type": []string{"text/html"}},
			Body:       io.NopCloser(bytes.NewBufferString(`<html><body><h1>Raw Content</h1></body></html>`)),
		}, nil
	})

	content, prefix, err := fetchURL(context.Background(), "http://example.com", "test-agent", true, "")
	suite.NoError(err)
	suite.Contains(prefix, "Content type text/html cannot be simplified to markdown")
	suite.Equal(`<html><body><h1>Raw Content</h1></body></html>`, content)
}

func (suite *ContentTestSuite) TestFetchURL_ErrorStatusCode() {
	suite.setMockHTTPClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 404,
			Body:       io.NopCloser(bytes.NewBufferString(`Not Found`)),
		}, nil
	})

	_, _, err := fetchURL(context.Background(), "http://example.com", "test-agent", false, "")
	suite.ErrorContains(err, "status code 404")
}

func TestContentSuite(t *testing.T) {
	suite.Run(t, new(ContentTestSuite))
}

type errReader int

func (errReader) Read(p []byte) (n int, err error) {
	return 0, fmt.Errorf("mock error")
}
func (errReader) Close() error {
	return nil
}

func (suite *ContentTestSuite) TestFetchURL_InvalidURL() {
	_, _, err := fetchURL(context.Background(), "http://%invalid", "test-agent", false, "")
	suite.Error(err)
}

func (suite *ContentTestSuite) TestFetchURL_ClientDoError() {
	suite.setMockHTTPClient(func(req *http.Request) (*http.Response, error) {
		return nil, fmt.Errorf("client error")
	})

	_, _, err := fetchURL(context.Background(), "http://example.com", "test-agent", false, "")
	suite.Error(err)
	suite.ErrorContains(err, "failed to fetch")
}

func (suite *ContentTestSuite) TestFetchURL_HTTPClientError() {
	// Trigger getHTTPClient error by providing an invalid proxy
	_, _, err := fetchURL(context.Background(), "http://example.com", "test-agent", false, "http://%invalid")
	suite.Error(err)
}

func (suite *ContentTestSuite) TestMin() {
	suite.Equal(5, min(5, 10))
	suite.Equal(5, min(10, 5))
	suite.Equal(5, min(5, 5))
}
