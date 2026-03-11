package fetch

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type HTTPTestSuite struct {
	suite.Suite
}

func (suite *HTTPTestSuite) TestHTTPClientFactory_NoProxy() {
	client, err := HTTPClientFactory("")
	suite.NoError(err)
	suite.NotNil(client)
	suite.Nil(client.Transport)
}

func (suite *HTTPTestSuite) TestHTTPClientFactory_ValidProxy() {
	client, err := HTTPClientFactory("http://proxy.example.com:8080")
	suite.NoError(err)
	suite.NotNil(client)
	suite.NotNil(client.Transport)
}

func (suite *HTTPTestSuite) TestHTTPClientFactory_InvalidProxy() {
	client, err := HTTPClientFactory("http://[::1]:namedport") // Invalid URL
	suite.Error(err)
	suite.Nil(client)
}

func TestHTTPSuite(t *testing.T) {
	suite.Run(t, new(HTTPTestSuite))
}
