package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/aqua777/krait"
	"github.com/aqua777/mcp-servers/examples/utils"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/sashabaranov/go-openai"
)

const (
	llmModel = "qwen3:0.6b" // Use a model available in Ollama
)

func runSequentialThinkingMCPServer(args []string) error {
	userQuery := strings.Join(args, " ")

	if len(userQuery) == 0 {
		return fmt.Errorf("usage: go run main.go [flags] \"Your complex query here (e.g. math or logic puzzle)\"")
	}

	ctx := context.Background()

	// 1. Setup the actual Transport (the "wire")
	// This starts the MCP server process as a child
	cmd := exec.Command("go", "run", "../../cmd/sequentialthinking/main.go")
	cmd.Stderr = os.Stderr

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}
	defer cmd.Process.Kill()

	transport := &mcp.IOTransport{
		Reader: stdout,
		Writer: stdin,
	}

	// 2. Initialize the Client
	mcpClient := mcp.NewClient(&mcp.Implementation{
		Name:    "my-go-chat-app",
		Version: "1.0.0",
	}, nil)

	// 3. Connect the client to the transport
	session, err := mcpClient.Connect(ctx, transport, nil)
	if err != nil {
		return err
	}
	defer session.Close()

	// 4. Discover Tools from MCP Server
	toolsResult, err := session.ListTools(ctx, nil)
	if err != nil {
		return fmt.Errorf("ListTools Error: %v", err)
	}
	fmt.Printf("Discovered tools: %d\n", len(toolsResult.Tools))

	// 5. Convert MCP Tools to OpenAI/Ollama Tools
	var ollamaTools []openai.Tool
	for _, t := range toolsResult.Tools {
		ollamaTools = append(ollamaTools, openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.InputSchema,
			},
		})
	}

	// 6. Configure Ollama Client (OpenAI-compatible)
	config := openai.DefaultConfig("")
	config.BaseURL = "http://host.docker.internal:11434/v1" // Standard Ollama local port
	llm := openai.NewClientWithConfig(config)

	messages := []openai.ChatCompletionMessage{
		{
			Role: openai.ChatMessageRoleSystem,
			Content: "You are a helpful assistant. You have a sequentialthinking tool. " +
				"Use it to break down complex problems step-by-step before answering. " +
				"Keep calling the tool with your thoughts until you are ready to give the final answer.",
		},
		{Role: openai.ChatMessageRoleUser, Content: userQuery},
	}

	// 7. Loop for LLM Tool Calls
	for {
		resp, err := llm.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
			Model:    llmModel,
			Messages: messages,
			Tools:    ollamaTools,
		})
		if err != nil {
			return fmt.Errorf("LLM Error: %v", err)
		}

		msg := resp.Choices[0].Message

		if len(msg.ToolCalls) == 0 {
			fmt.Println("\n🤖 Final Response:", msg.Content)
			break
		}

		// Add assistant message to history to maintain context
		messages = append(messages, msg)

		for _, tc := range msg.ToolCalls {
			fmt.Printf("\n🔧 LLM calling tool: %s\n", tc.Function.Name)

			var args map[string]interface{}
			json.Unmarshal([]byte(tc.Function.Arguments), &args)
			fmt.Printf("Arguments: %s\n", utils.JsonStr(args))

			// Execute the tool via MCP
			callResp, err := session.CallTool(ctx, &mcp.CallToolParams{
				Name:      tc.Function.Name,
				Arguments: args,
			})
			if err != nil {
				return fmt.Errorf("MCP Call Error: %v", err)
			}

			toolResultContent, _ := json.Marshal(callResp.Content)
			fmt.Printf("Tool Result: %s\n", toolResultContent)

			messages = append(messages, openai.ChatCompletionMessage{
				Role:       openai.ChatMessageRoleTool,
				Content:    string(toolResultContent),
				ToolCallID: tc.ID,
			})
		}
	}
	return nil
}

func main() {
	app := krait.App("example-sequentialthinking", "An example chat app that uses the sequentialthinking MCP server", "An example chat app that uses the sequentialthinking MCP server").
		WithConfig("", "config", "c", "APP_CONFIG").
		WithRun(runSequentialThinkingMCPServer)

	if err := app.Execute(); err != nil {
		log.Fatal(err)
	}
}
