package time

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/suite"
)

type TimeServerTestSuite struct {
	suite.Suite
	ts *TimeServer
}

func (s *TimeServerTestSuite) SetupTest() {
	s.ts = &TimeServer{}
}

func (s *TimeServerTestSuite) callTool(name string, args map[string]interface{}) (*mcp.CallToolResult, error) {
	argsBytes, err := json.Marshal(args)
	s.Require().NoError(err)

	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      name,
			Arguments: argsBytes,
		},
	}

	if name == ToolGetCurrentTime {
		return s.ts.handleGetCurrentTime(context.Background(), req)
	} else if name == ToolConvertTime {
		return s.ts.handleConvertTime(context.Background(), req)
	}

	return nil, nil
}

func (s *TimeServerTestSuite) TestNewServer() {
	srv, err := NewServer(context.Background(), nil)
	s.Require().NoError(err)
	s.NotNil(srv)
}

func (s *TimeServerTestSuite) TestGetCurrentTime() {
	args := map[string]interface{}{
		"timezone": "America/New_York",
	}

	result, err := s.callTool(ToolGetCurrentTime, args)
	s.Require().NoError(err)
	s.Require().NotNil(result)
	s.Require().False(result.IsError)

	var timeResult TimeResult
	textContent := result.Content[0].(*mcp.TextContent)
	err = json.Unmarshal([]byte(textContent.Text), &timeResult)
	s.Require().NoError(err)

	s.Equal("America/New_York", timeResult.Timezone)
	s.NotEmpty(timeResult.Datetime)
	s.NotEmpty(timeResult.DayOfWeek)
}

func (s *TimeServerTestSuite) TestGetCurrentTime_MissingTimezone() {
	args := map[string]interface{}{}

	result, err := s.callTool(ToolGetCurrentTime, args)
	s.Require().NoError(err)
	s.Require().True(result.IsError)
}

func (s *TimeServerTestSuite) TestGetCurrentTime_InvalidTimezone() {
	args := map[string]interface{}{
		"timezone": "Invalid/Timezone",
	}

	result, err := s.callTool(ToolGetCurrentTime, args)
	s.Require().NoError(err)
	s.Require().True(result.IsError)
}

func (s *TimeServerTestSuite) TestConvertTime() {
	args := map[string]interface{}{
		"source_timezone": "America/New_York",
		"time":            "10:00",
		"target_timezone": "Europe/London",
	}

	result, err := s.callTool(ToolConvertTime, args)
	s.Require().NoError(err)
	s.Require().NotNil(result)
	s.Require().False(result.IsError)

	var conversionResult TimeConversionResult
	textContent := result.Content[0].(*mcp.TextContent)
	err = json.Unmarshal([]byte(textContent.Text), &conversionResult)
	s.Require().NoError(err)

	s.Equal("America/New_York", conversionResult.Source.Timezone)
	s.Equal("Europe/London", conversionResult.Target.Timezone)
	s.NotEmpty(conversionResult.TimeDifference)
}

func (s *TimeServerTestSuite) TestConvertTime_MissingArgs() {
	argsList := []map[string]interface{}{
		{},
		{"source_timezone": "UTC"},
		{"source_timezone": "UTC", "time": "12:00"},
	}

	for _, args := range argsList {
		result, err := s.callTool(ToolConvertTime, args)
		s.Require().NoError(err)
		s.Require().True(result.IsError)
	}
}

func (s *TimeServerTestSuite) TestConvertTime_InvalidFormat() {
	args := map[string]interface{}{
		"source_timezone": "America/New_York",
		"time":            "10:00 AM", // Invalid format, should be HH:MM
		"target_timezone": "Europe/London",
	}

	result, err := s.callTool(ToolConvertTime, args)
	s.Require().NoError(err)
	s.Require().True(result.IsError)
}

func (s *TimeServerTestSuite) TestConvertTime_FractionalOffset() {
	// Kathmandu is UTC+5:45
	args := map[string]interface{}{
		"source_timezone": "UTC",
		"time":            "12:00",
		"target_timezone": "Asia/Kathmandu",
	}

	result, err := s.callTool(ToolConvertTime, args)
	s.Require().NoError(err)
	s.Require().NotNil(result)
	s.Require().False(result.IsError)

	var conversionResult TimeConversionResult
	textContent := result.Content[0].(*mcp.TextContent)
	err = json.Unmarshal([]byte(textContent.Text), &conversionResult)
	s.Require().NoError(err)

	s.Equal("+5.75h", conversionResult.TimeDifference)
}

func TestTimeServerSuite(t *testing.T) {
	suite.Run(t, new(TimeServerTestSuite))
}
