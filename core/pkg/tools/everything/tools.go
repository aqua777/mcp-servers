package everything

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func registerTools(server *mcp.Server) {
	// echo tool
	server.AddTool(&mcp.Tool{
		Name:        "echo",
		Description: "Echoes back the provided message",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"message": map[string]any{
					"type":        "string",
					"description": "The message to echo",
				},
			},
			"required": []string{"message"},
		},
	}, handleEcho)

	// get-env tool
	server.AddTool(&mcp.Tool{
		Name:        "get-env",
		Description: "Returns all environment variables from the running process as a JSON string",
		InputSchema: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
	}, handleGetEnv)

	// get-sum tool
	server.AddTool(&mcp.Tool{
		Name:        "get-sum",
		Description: "Calculates the sum of two numbers",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"a": map[string]any{
					"type":        "number",
					"description": "First number",
				},
				"b": map[string]any{
					"type":        "number",
					"description": "Second number",
				},
			},
			"required": []string{"a", "b"},
		},
	}, handleGetSum)
}

func handleEcho(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(request.Params.Arguments, &args); err != nil {
		return handleError(err)
	}

	return handleSuccess(fmt.Sprintf("Echo: %s", args.Message))
}

func handleGetEnv(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	envVars := make(map[string]string)
	for _, env := range os.Environ() {
		for i := 0; i < len(env); i++ {
			if env[i] == '=' {
				envVars[env[:i]] = env[i+1:]
				break
			}
		}
	}

	b, _ := json.MarshalIndent(envVars, "", "  ")

	return handleSuccess(string(b))
}

func handleGetSum(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args struct {
		A float64 `json:"a"`
		B float64 `json:"b"`
	}
	if err := json.Unmarshal(request.Params.Arguments, &args); err != nil {
		return handleError(err)
	}

	sum := args.A + args.B

	return handleSuccess(sum)
}

func handleSuccess(data interface{}) (*mcp.CallToolResult, error) {
	var text string
	switch v := data.(type) {
	case string:
		text = v
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return handleError(err)
		}
		text = string(b)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: text,
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
