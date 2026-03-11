package everything

import (
"context"
"encoding/json"
"fmt"

"github.com/modelcontextprotocol/go-sdk/mcp"
)

func registerTriggerSamplingRequestTool(server *mcp.Server) {
	server.AddTool(&mcp.Tool{
		Name:        "trigger-sampling-request",
		Description: "Trigger a Request from the Server for LLM Sampling",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"prompt": map[string]any{
					"type":        "string",
					"description": "The prompt to send to the LLM",
				},
				"maxTokens": map[string]any{
					"type":        "number",
					"description": "Maximum number of tokens to generate",
					"default":     100,
				},
			},
			"required": []string{"prompt"},
		},
	}, handleTriggerSamplingRequest)
}

func handleTriggerSamplingRequest(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	session := request.Session
	if session == nil {
		return handleSuccess("No active session to trigger sampling.")
	}

	// Check if client supports sampling
	iparams := session.InitializeParams()
	if iparams == nil || iparams.Capabilities == nil || iparams.Capabilities.Sampling == nil {
		return handleSuccess("The client does not support sampling.")
	}

	var args struct {
		Prompt    string  `json:"prompt"`
		MaxTokens float64 `json:"maxTokens"`
	}
	if err := json.Unmarshal(request.Params.Arguments, &args); err != nil {
		return handleError(err)
	}
	if args.MaxTokens == 0 {
		args.MaxTokens = 100
	}

	res, err := session.CreateMessage(ctx, &mcp.CreateMessageParams{
		Messages: []*mcp.SamplingMessage{
			{
				Role: mcp.Role("user"),
				Content: &mcp.TextContent{
					Text: fmt.Sprintf("Resource trigger-sampling-request context: %s", args.Prompt),
				},
			},
		},
		SystemPrompt: "You are a helpful test server.",
		MaxTokens:    int64(args.MaxTokens),
	})
	if err != nil {
		return handleError(fmt.Errorf("failed to get sampling response from client: %w", err))
	}

	b, _ := json.MarshalIndent(res, "", "  ")
	return handleSuccess(fmt.Sprintf("LLM sampling result: \n%s", string(b)))
}
