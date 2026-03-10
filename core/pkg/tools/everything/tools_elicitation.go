package everything

import (
"context"
"encoding/json"
"fmt"

"github.com/modelcontextprotocol/go-sdk/mcp"
)

func registerTriggerElicitationRequestTool(server *mcp.Server) {
	server.AddTool(&mcp.Tool{
		Name:        "trigger-elicitation-request",
		Description: "Triggers an elicitation request to the client to ask for user input.",
		InputSchema: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
	}, handleTriggerElicitationRequest)
}

func handleTriggerElicitationRequest(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	session := request.Session
	if session == nil {
		return handleSuccess("No active session to trigger elicitation.")
	}

	// Check if client supports elicitation
	iparams := session.InitializeParams()
	if iparams == nil || iparams.Capabilities == nil || iparams.Capabilities.Elicitation == nil {
		return handleSuccess("The client does not support elicitation.")
	}

	res, err := session.Elicit(ctx, &mcp.ElicitParams{
		Message: "Please provide inputs for the following fields:",
		RequestedSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name": map[string]any{
					"title":       "String",
					"type":        "string",
					"description": "Your full, legal name",
				},
				"check": map[string]any{
					"title":       "Boolean",
					"type":        "boolean",
					"description": "Agree to the terms and conditions",
				},
				"integer": map[string]any{
					"title":       "Integer",
					"type":        "integer",
					"description": "Your favorite integer",
					"minimum":     1,
					"maximum":     100,
					"default":     42,
				},
			},
			"required": []string{"name"},
		},
	})
	if err != nil {
		return handleError(fmt.Errorf("failed to get elicitation response from client: %w", err))
	}

	if res == nil {
		return handleSuccess("The client returned an empty elicitation response.")
	}

	var content []mcp.Content
	
	switch res.Action {
	case "accept":
		content = append(content, &mcp.TextContent{Text: "✅ User provided the requested information!"})
		if len(res.Content) > 0 {
			b, _ := json.MarshalIndent(res.Content, "", "  ")
			content = append(content, &mcp.TextContent{Text: fmt.Sprintf("User inputs:\n%s", string(b))})
		}
	case "decline":
		content = append(content, &mcp.TextContent{Text: "❌ User declined to provide the requested information."})
	case "cancel":
		content = append(content, &mcp.TextContent{Text: "⚠️ User cancelled the elicitation dialog."})
	}
	
	b, _ := json.MarshalIndent(res, "", "  ")
	content = append(content, &mcp.TextContent{Text: fmt.Sprintf("\nRaw result: %s", string(b))})

	return &mcp.CallToolResult{Content: content}, nil
}
