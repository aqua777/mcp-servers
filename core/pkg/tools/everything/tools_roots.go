package everything

import (
"context"
"fmt"

"github.com/modelcontextprotocol/go-sdk/mcp"
)

func registerGetRootsListTool(server *mcp.Server) {
	server.AddTool(&mcp.Tool{
		Name:        "get-roots-list",
		Description: "Lists the current MCP roots provided by the client. Demonstrates the roots protocol capability even though this server doesn't access files.",
		InputSchema: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
	}, handleGetRootsList)
}

func handleGetRootsList(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	session := request.Session
	if session == nil {
		return handleSuccess("No active session to retrieve roots from.")
	}

	// Check if client supports roots
	iparams := session.InitializeParams()
	if iparams == nil || iparams.Capabilities == nil {
		return handleSuccess("The client does not support the roots capability.")
	}

	res, err := session.ListRoots(ctx, nil)
	if err != nil {
		return handleError(fmt.Errorf("failed to get roots from client: %w", err))
	}

	if res == nil || len(res.Roots) == 0 {
		return handleSuccess("The client supports roots but no roots are currently configured.\n\nThis could mean:\n1. The client hasn't provided any roots yet\n2. The client provided an empty roots list\n3. The roots configuration is still being loaded")
	}

	rootsList := ""
	for i, root := range res.Roots {
		name := root.Name
		if name == "" {
			name = "Unnamed Root"
		}
		rootsList += fmt.Sprintf("%d. %s\n   URI: %s\n\n", i+1, name, root.URI)
	}

	responseText := fmt.Sprintf("Current MCP Roots (%d total):\n\n%s\nNote: This server demonstrates the roots protocol capability but doesn't actually access files. The roots are provided by the MCP client and can be used by servers that need file system access.", len(res.Roots), rootsList)

	return handleSuccess(responseText)
}
