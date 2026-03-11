package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	fmt.Println("Starting everything MCP server demo...")

	ctx := context.Background()

	cmd := exec.CommandContext(ctx, "go", "run", "../../cmd/everything")
	cmd.Stderr = os.Stderr

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatalf("Failed to create stdout pipe: %v", err)
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		log.Fatalf("Failed to create stdin pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

	go func() {
		cmd.Wait()
	}()

	fmt.Println("Server started. Initializing client...")

	transport := &mcp.IOTransport{
		Reader: stdout,
		Writer: stdin,
	}

	mcpClient := mcp.NewClient(&mcp.Implementation{
		Name:    "demo-client",
		Version: "1.0.0",
	}, nil)

	session, err := mcpClient.Connect(ctx, transport, nil)
	if err != nil {
		log.Fatalf("Failed to connect client: %v", err)
	}
	defer session.Close()

	fmt.Println("Client initialized successfully.")

	toolsResult, err := session.ListTools(ctx, nil)
	if err != nil {
		log.Fatalf("Failed to list tools: %v", err)
	}

	fmt.Println("Available tools:")
	for _, tool := range toolsResult.Tools {
		fmt.Printf("- %s: %s\n", tool.Name, tool.Description)
	}

	fmt.Println("\nCalling echo tool...")
	callResp, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "echo",
		Arguments: map[string]interface{}{
			"message": "Hello from the demo app!",
		},
	})
	if err != nil {
		log.Fatalf("Failed to call echo tool: %v", err)
	}

	toolResultContent, _ := json.MarshalIndent(callResp.Content, "", "  ")
	fmt.Printf("Echo result:\n%s\n", toolResultContent)

	fmt.Println("\nDemo completed.")
}
