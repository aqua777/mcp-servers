package everything

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func registerPrompts(server *mcp.Server) {
	// simple-prompt
	server.AddPrompt(&mcp.Prompt{
		Name:        "simple-prompt",
		Description: "A prompt with no arguments",
	}, handleSimplePrompt)

	// args-prompt
	server.AddPrompt(&mcp.Prompt{
		Name:        "args-prompt",
		Description: "A prompt with arguments",
		Arguments: []*mcp.PromptArgument{
			{
				Name:        "city",
				Description: "City name",
				Required:    true,
			},
			{
				Name:        "state",
				Description: "State name",
				Required:    false,
			},
		},
	}, handleArgsPrompt)
}

func handleSimplePrompt(ctx context.Context, request *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	return &mcp.GetPromptResult{
		Description: "Simple Prompt",
		Messages: []*mcp.PromptMessage{
			{
				Role: "user",
				Content: &mcp.TextContent{
					Text: "This is a simple prompt without arguments.",
				},
			},
		},
	}, nil
}

func handleArgsPrompt(ctx context.Context, request *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	args := request.Params.Arguments
	city, ok := args["city"]
	if !ok || city == "" {
		return nil, fmt.Errorf("city argument is required")
	}

	state := args["state"]
	var text string
	if state != "" {
		text = fmt.Sprintf("Tell me about the weather in %s, %s.", city, state)
	} else {
		text = fmt.Sprintf("Tell me about the weather in %s.", city)
	}

	return &mcp.GetPromptResult{
		Description: "Args Prompt",
		Messages: []*mcp.PromptMessage{
			{
				Role: "user",
				Content: &mcp.TextContent{
					Text: text,
				},
			},
		},
	}, nil
}
