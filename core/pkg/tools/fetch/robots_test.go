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

type RobotsTestSuite struct {
	suite.Suite
	originalHTTPClientFactory func(proxyURL string) (*http.Client, error)
}

func (suite *RobotsTestSuite) SetupSuite() {
	suite.originalHTTPClientFactory = HTTPClientFactory
}

func (suite *RobotsTestSuite) TearDownSuite() {
	HTTPClientFactory = suite.originalHTTPClientFactory
}

func (suite *RobotsTestSuite) setMockHTTPClient(handler func(req *http.Request) (*http.Response, error)) {
	HTTPClientFactory = func(proxyURL string) (*http.Client, error) {
		return &http.Client{
			Transport: &mockRoundTripper{handler: handler},
		}, nil
	}
}

func (suite *RobotsTestSuite) TestGetRobotsTxtURL() {
	tests := []struct {
		name     string
		url      string
		expected string
		wantErr  bool
	}{
		{
			name:     "simple URL",
			url:      "https://example.com/page",
			expected: "https://example.com/robots.txt",
			wantErr:  false,
		},
		{
			name:     "URL with path",
			url:      "https://example.com/some/deep/path/page.html",
			expected: "https://example.com/robots.txt",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			result, err := getRobotsTxtURL(tt.url)
			if tt.wantErr {
				suite.Error(err)
			} else {
				suite.NoError(err)
				suite.Equal(tt.expected, result)
			}
		})
	}
}

func (suite *RobotsTestSuite) TestProcessRobotsTxt() {
	input := `User-agent: *
# This is a comment
Disallow: /admin
# Another comment
Allow: /public`
	expected := `User-agent: *
Disallow: /admin
Allow: /public`
	result := processRobotsTxt(input)
	suite.Equal(expected, result)
}

func (suite *RobotsTestSuite) TestCheckMayAutonomouslyFetchURL_Allowed() {
	suite.setMockHTTPClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewBufferString("User-agent: *\nAllow: /")),
		}, nil
	})

	err := checkMayAutonomouslyFetchURL(context.Background(), "http://example.com/test", "test-agent", "")
	suite.NoError(err)
}

func (suite *RobotsTestSuite) TestCheckMayAutonomouslyFetchURL_Disallowed() {
	suite.setMockHTTPClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewBufferString("User-agent: *\nDisallow: /")),
		}, nil
	})

	err := checkMayAutonomouslyFetchURL(context.Background(), "http://example.com/test", "test-agent", "")
	suite.ErrorContains(err, "autonomous fetching of this page is not allowed")
}

func (suite *RobotsTestSuite) TestCheckMayAutonomouslyFetchURL_Unauthorized() {
	suite.setMockHTTPClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 401,
		}, nil
	})

	err := checkMayAutonomouslyFetchURL(context.Background(), "http://example.com/test", "test-agent", "")
	suite.ErrorContains(err, "assuming that autonomous fetching is not allowed")
}

func (suite *RobotsTestSuite) TestCheckMayAutonomouslyFetchURL_404() {
	suite.setMockHTTPClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 404,
		}, nil
	})

	// 404 means no robots.txt, which implies allowed
	err := checkMayAutonomouslyFetchURL(context.Background(), "http://example.com/test", "test-agent", "")
	suite.NoError(err)
}

func TestRobotsSuite(t *testing.T) {
	suite.Run(t, new(RobotsTestSuite))
}

func (suite *RobotsTestSuite) TestGetRobotsTxtURL_InvalidURL() {
	_, err := getRobotsTxtURL("http://%invalid")
	suite.Error(err)
}

func (suite *RobotsTestSuite) TestCheckMayAutonomouslyFetchURL_InvalidURL() {
	err := checkMayAutonomouslyFetchURL(context.Background(), "http://%invalid", "test-agent", "")
	suite.Error(err)
}

func (suite *RobotsTestSuite) TestCheckMayAutonomouslyFetchURL_ClientError() {
	suite.setMockHTTPClient(func(req *http.Request) (*http.Response, error) {
		return nil, fmt.Errorf("client error")
	})

	err := checkMayAutonomouslyFetchURL(context.Background(), "http://example.com/test", "test-agent", "")
	suite.Error(err)
	suite.ErrorContains(err, "failed to fetch robots.txt")
}

func (suite *RobotsTestSuite) TestCheckMayAutonomouslyFetchURL_HTTPClientError() {
	HTTPClientFactory = suite.originalHTTPClientFactory
	err := checkMayAutonomouslyFetchURL(context.Background(), "http://example.com/test", "test-agent", "http://%invalid")
	suite.Error(err)
	suite.ErrorContains(err, "invalid proxy URL")
}
