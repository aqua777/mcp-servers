package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/aqua777/krait"
	"github.com/aqua777/mcp-servers/examples/utils"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/sashabaranov/go-openai"
)

const (
	llmModel = "qwen3:0.6b"
)

func runMemoryMCPServer(args []string) error {
	memoryFilePath := krait.GetString("app.memory-file-path")
	userQuery := strings.Join(args, " ")

	if len(userQuery) == 0 {
		return fmt.Errorf("usage: go run main.go [flags] \"Your query here\"")
	}

	// Default to memory.jsonl in current directory if not specified
	if len(memoryFilePath) == 0 {
		memoryPath, err := filepath.Abs(filepath.Join(".", "memory.jsonl"))
		if err != nil {
			return fmt.Errorf("error resolving memory file path: %w", err)
		}
		memoryFilePath = memoryPath
	}

	ctx := context.Background()

	// 1. Setup the actual Transport (the "wire")
	// Run the memory server with the memory file path
	cmd := exec.Command("go", "run", "../../cmd/memory-mcp/main.go", "--memory-file-path", memoryFilePath)
	cmd.Env = os.Environ()
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
	fmt.Printf("Discovered tools: %s\n", utils.JsonStr(toolsResult.Tools))

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
		{Role: openai.ChatMessageRoleSystem, Content: "Always use MCP tools to answer; search the memory graph first."},
		{Role: openai.ChatMessageRoleUser, Content: userQuery},
	}

	// 7. Initial Call to LLM
	resp, err := llm.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:    llmModel,
		Messages: messages,
		Tools:    ollamaTools,
	})
	if err != nil {
		return fmt.Errorf("LLM Error: %v", err)
	}

	msg := resp.Choices[0].Message
	fmt.Printf("LLM Response: %s\n", utils.JsonStr(msg))

	// 8. Handle Tool Calls
	if len(msg.ToolCalls) > 0 {
		for _, tc := range msg.ToolCalls {
			fmt.Printf("🔧 LLM calling tool: %s\n", tc.Function.Name)

			// Execute the tool via MCP
			var args map[string]interface{}
			json.Unmarshal([]byte(tc.Function.Arguments), &args)

			callResp, err := session.CallTool(ctx, &mcp.CallToolParams{
				Name:      tc.Function.Name,
				Arguments: args,
			})
			if err != nil {
				return fmt.Errorf("MCP Call Error: %v", err)
			}

			// Capture the tool result
			toolResultContent, _ := json.Marshal(callResp.Content)
			fmt.Printf("Tool Result:\n---\n%s\n---\n", toolResultContent)

			// Add the LLM's tool call and the tool's result to conversation history
			messages = append(messages, msg)
			messages = append(messages, openai.ChatCompletionMessage{
				Role:       openai.ChatMessageRoleTool,
				Content:    string(toolResultContent),
				ToolCallID: tc.ID,
			})
		}

		// Final LLM Answer with Tool Data
		finalResp, _ := llm.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
			Model:    llmModel,
			Messages: messages,
		})
		fmt.Println("\n🤖 Response:", finalResp.Choices[0].Message.Content)
	} else {
		fmt.Println("\n🤖 Response:", msg.Content)
	}
	return nil
}

func main() {
	app := krait.App("example-memory", "An example chat app that uses the memory MCP server", "An example chat app that uses the memory MCP server").
		WithConfig("", "config", "c", "APP_CONFIG").
		WithStringP("app.memory-file-path", "Path to the memory.jsonl file", "memory-file-path", "m", "MEMORY_FILE_PATH", "").
		WithRun(runMemoryMCPServer)

	if err := app.Execute(); err != nil {
		log.Fatal(err)
	}
}
