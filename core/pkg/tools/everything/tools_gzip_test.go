package everything

import (
"testing"

"github.com/modelcontextprotocol/go-sdk/mcp"
"github.com/stretchr/testify/suite"
)

type GzipTestSuite struct {
	suite.Suite
	server *mcp.Server
}

func (s *GzipTestSuite) SetupTest() {
	s.server = mcp.NewServer(&mcp.Implementation{
		Name:    "mcp-everything",
		Version: "0.1.0",
	}, &mcp.ServerOptions{})
	
	registerGZipFileAsResourceTool(s.server)
}

func (s *GzipTestSuite) TestGzipConfig() {
	s.Equal(10*1024*1024, gzipMaxFetchSize)
	s.Equal(30000, gzipMaxFetchTimeMillis)
	s.Empty(gzipAllowedDomains)
}

func TestGzipSuite(t *testing.T) {
	suite.Run(t, new(GzipTestSuite))
}
