package time

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aqua777/mcp-servers/common"
	"github.com/aqua777/mcp-servers/core/pkg/runtime"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func init() {
	runtime.Register(common.MCP_Time, NewServer)
}

type TimeServer struct {
	server *mcp.Server
}

func NewServer(ctx context.Context, opts any) (*mcp.Server, error) {
	ts := &TimeServer{}
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "mcp-time",
		Version: "1.0.0",
	}, &mcp.ServerOptions{})
	ts.server = server

	// get_current_time
	server.AddTool(&mcp.Tool{
		Name:        ToolGetCurrentTime,
		Description: "Get current time in a specific timezone",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"timezone": map[string]any{
					"type":        "string",
					"description": "IANA timezone name (e.g., 'America/New_York', 'Europe/London'). Use local timezone if no timezone provided.",
				},
			},
			"required": []string{"timezone"},
		},
	}, ts.handleGetCurrentTime)

	// convert_time
	server.AddTool(&mcp.Tool{
		Name:        ToolConvertTime,
		Description: "Convert time between timezones",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"source_timezone": map[string]any{
					"type":        "string",
					"description": "Source IANA timezone name (e.g., 'America/New_York', 'Europe/London'). Use local timezone if no source timezone provided.",
				},
				"time": map[string]any{
					"type":        "string",
					"description": "Time to convert in 24-hour format (HH:MM)",
				},
				"target_timezone": map[string]any{
					"type":        "string",
					"description": "Target IANA timezone name (e.g., 'Asia/Tokyo', 'America/San_Francisco'). Use local timezone if no target timezone provided.",
				},
			},
			"required": []string{"source_timezone", "time", "target_timezone"},
		},
	}, ts.handleConvertTime)

	return server, nil
}

func (ts *TimeServer) handleGetCurrentTime(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args struct {
		Timezone string `json:"timezone"`
	}
	if err := json.Unmarshal(request.Params.Arguments, &args); err != nil {
		return handleError(err)
	}

	if args.Timezone == "" {
		return handleError(fmt.Errorf("Missing required argument: timezone"))
	}

	loc, err := time.LoadLocation(args.Timezone)
	if err != nil {
		return handleError(fmt.Errorf("Invalid timezone: %v", err))
	}

	now := time.Now().In(loc)
	result := formatTimeResult(now, args.Timezone)
	return handleSuccess(result)
}

func (ts *TimeServer) handleConvertTime(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args struct {
		SourceTimezone string `json:"source_timezone"`
		Time           string `json:"time"`
		TargetTimezone string `json:"target_timezone"`
	}
	if err := json.Unmarshal(request.Params.Arguments, &args); err != nil {
		return handleError(err)
	}

	if args.SourceTimezone == "" {
		return handleError(fmt.Errorf("Missing required argument: source_timezone"))
	}
	if args.Time == "" {
		return handleError(fmt.Errorf("Missing required argument: time"))
	}
	if args.TargetTimezone == "" {
		return handleError(fmt.Errorf("Missing required argument: target_timezone"))
	}

	sourceLoc, err := time.LoadLocation(args.SourceTimezone)
	if err != nil {
		return handleError(fmt.Errorf("Invalid source timezone: %v", err))
	}

	targetLoc, err := time.LoadLocation(args.TargetTimezone)
	if err != nil {
		return handleError(fmt.Errorf("Invalid target timezone: %v", err))
	}

	parsedTime, err := time.Parse("15:04", args.Time)
	if err != nil {
		return handleError(fmt.Errorf("Invalid time format. Expected HH:MM [24-hour format]"))
	}

	now := time.Now().In(sourceLoc)
	sourceTime := time.Date(now.Year(), now.Month(), now.Day(), parsedTime.Hour(), parsedTime.Minute(), 0, 0, sourceLoc)
	targetTime := sourceTime.In(targetLoc)

	_, sourceOffset := sourceTime.Zone()
	_, targetOffset := targetTime.Zone()
	diffSeconds := targetOffset - sourceOffset
	hoursDiff := float64(diffSeconds) / 3600.0

	var timeDiffStr string
	if hoursDiff == float64(int(hoursDiff)) {
		timeDiffStr = fmt.Sprintf("%+.1fh", hoursDiff)
	} else {
		timeDiffStr = fmt.Sprintf("%+.2fh", hoursDiff)
	}

	result := TimeConversionResult{
		Source:         formatTimeResult(sourceTime, args.SourceTimezone),
		Target:         formatTimeResult(targetTime, args.TargetTimezone),
		TimeDifference: timeDiffStr,
	}

	return handleSuccess(result)
}

func handleSuccess(data interface{}) (*mcp.CallToolResult, error) {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return handleError(err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: string(b),
			},
		},
	}, nil
}

func handleError(err error) (*mcp.CallToolResult, error) {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: fmt.Sprintf("Error: %v", err),
			},
		},
		IsError: true,
	}, nil
}
