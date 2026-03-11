package resources

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type TemplatesTestSuite struct {
	suite.Suite
}

func TestTemplatesSuite(t *testing.T) {
	suite.Run(t, new(TemplatesTestSuite))
}

func (s *TemplatesTestSuite) TestTextResource() {
	uri := TextResourceUri(1)
	s.Require().Equal("demo://resource/dynamic/text/1", uri)

	res := TextResource(uri, 1)
	s.Require().Equal(uri, res.URI)
	s.Require().Equal("text/plain", res.MIMEType)
	s.Require().Contains(res.Text, "dynamically generated text resource 1")
	s.Require().Contains(res.Text, time.Now().Format("2006")) // rudimentary check for timestamp
}

func (s *TemplatesTestSuite) TestBlobResource() {
	uri := BlobResourceUri(2)
	s.Require().Equal("demo://resource/dynamic/blob/2", uri)

	res := BlobResource(uri, 2)
	s.Require().Equal(uri, res.URI)
	s.Require().Equal("application/octet-stream", res.MIMEType)
	s.Require().NotEmpty(res.Blob)
}
