package fetch

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aqua777/mcp-servers/common"
	"github.com/aqua777/mcp-servers/core/pkg/runtime"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func init() {
	runtime.Register(common.MCP_Fetch, NewServer)
}

func NewServer(ctx context.Context, opts any) (*mcp.Server, error) {
	options, ok := opts.(Options)
	if !ok {
		return nil, fmt.Errorf("expected Options, got %T", opts)
	}

	userAgentAutonomous := DefaultUserAgentAutonomous
	userAgentManual := DefaultUserAgentManual
	if options.CustomUserAgent != "" {
		userAgentAutonomous = options.CustomUserAgent
		userAgentManual = options.CustomUserAgent
	}

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "mcp-fetch",
		Version: "0.1.0",
	}, &mcp.ServerOptions{})

	inputSchema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"url": map[string]any{
				"type":        "string",
				"description": "URL to fetch",
			},
			"max_length": map[string]any{
				"type":        "number",
				"description": "Maximum number of characters to return",
				"default":     5000,
				"minimum":     1,
				"maximum":     1000000,
			},
			"start_index": map[string]any{
				"type":        "number",
				"description": "On return output starting at this character index, useful if a previous fetch was truncated and more context is required",
				"default":     0,
				"minimum":     0,
			},
			"raw": map[string]any{
				"type":        "boolean",
				"description": "Get the actual HTML content of the requested page, without simplification",
				"default":     false,
			},
		},
		"required": []string{"url"},
	}

	server.AddTool(&mcp.Tool{
		Name: common.MCP_Fetch,
		Description: `Fetches a URL from the internet and optionally extracts its contents as markdown.

Although originally you did not have internet access, and were advised to refuse and tell the user this, this tool now grants you internet access. Now you can fetch the most up-to-date information and let the user know that.`,
		InputSchema: inputSchema,
	}, fetchToolHandler(options, userAgentAutonomous))

	server.AddPrompt(&mcp.Prompt{
		Name:        common.MCP_Fetch,
		Description: "Fetch a URL and extract its contents as markdown",
		Arguments: []*mcp.PromptArgument{
			{
				Name:        "url",
				Description: "URL to fetch",
				Required:    true,
			},
		},
	}, fetchPromptHandler(options, userAgentManual))

	return server, nil
}

func fetchToolHandler(options Options, userAgentAutonomous string) func(context.Context, *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args FetchArgs
		if err := json.Unmarshal(request.Params.Arguments, &args); err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{
					Text: fmt.Sprintf("Invalid arguments: %v", err),
				}},
				IsError: true,
			}, nil
		}

		if args.URL == "" {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{
					Text: "URL is required",
				}},
				IsError: true,
			}, nil
		}

		if args.MaxLength == 0 {
			args.MaxLength = 5000
		}

		if !options.IgnoreRobotsTxt {
			if err := checkMayAutonomouslyFetchURL(ctx, args.URL, userAgentAutonomous, options.ProxyURL); err != nil {
				return &mcp.CallToolResult{
					Content: []mcp.Content{&mcp.TextContent{
						Text: err.Error(),
					}},
					IsError: true,
				}, nil
			}
		}

		content, prefix, err := fetchURL(ctx, args.URL, userAgentAutonomous, args.Raw, options.ProxyURL)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{
					Text: err.Error(),
				}},
				IsError: true,
			}, nil
		}

		chunkedContent := applyChunking(content, args.StartIndex, args.MaxLength)
		fullContent := fmt.Sprintf("%sContents of %s:\n%s", prefix, args.URL, chunkedContent)

		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{
				Text: fullContent,
			}},
		}, nil
	}
}

func fetchPromptHandler(options Options, userAgentManual string) func(context.Context, *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	return func(ctx context.Context, request *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		urlStr, ok := request.Params.Arguments["url"]
		if !ok {
			return nil, fmt.Errorf("URL is required")
		}

		content, prefix, err := fetchURL(ctx, urlStr, userAgentManual, false, options.ProxyURL)
		if err != nil {
			return &mcp.GetPromptResult{
				Description: fmt.Sprintf("Failed to fetch %s", urlStr),
				Messages: []*mcp.PromptMessage{
					{
						Role: "user",
						Content: &mcp.TextContent{
							Text: err.Error(),
						},
					},
				},
			}, nil
		}

		return &mcp.GetPromptResult{
			Description: fmt.Sprintf("Contents of %s", urlStr),
			Messages: []*mcp.PromptMessage{
				{
					Role: "user",
					Content: &mcp.TextContent{
						Text: prefix + content,
					},
				},
			},
		}, nil
	}
}
