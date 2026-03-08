package main

import (
    "context"
	"encoding/json"
    "fmt"
    "os"
    "os/exec"
    "log"
    "strings"
    "github.com/modelcontextprotocol/go-sdk/mcp"
    "github.com/sashabaranov/go-openai"
)

const (
	llmModel = "qwen3:0.6b"
)

func jsonStr(v interface{}) string {
	b, _ := json.MarshalIndent(v, "", "  ")
	return string(b)
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: go run main.go \"Your query here\"")
	}
	userQuery := strings.Join(os.Args[1:], " ")
	ctx := context.Background()

    // 1. Setup the actual Transport (the "wire")
    // In Go, this starts the npx process as a child
    cmd := exec.Command("go", "run", "../../cmd/fetch/main.go") // ("npx", "-y", "@modelcontextprotocol/server-fetch")
	cmd.Stderr = os.Stderr

	stdout, err := cmd.StdoutPipe()
	if err != nil { log.Fatal(err) }
	stdin, err := cmd.StdinPipe()
	if err != nil { log.Fatal(err) }

	if err := cmd.Start(); err != nil { log.Fatal(err) }
	defer cmd.Process.Kill()

    transport := &mcp.IOTransport{
		Reader: stdout,
		Writer: stdin,
	}

    // 2. Initialize the Client
    // The implementation details (name/version) tell the server who we are
    mcpClient := mcp.NewClient(&mcp.Implementation{
        Name:    "my-go-chat-app",
        Version: "1.0.0",
    }, nil)

    // 3. Connect the client to the transport
    session, err := mcpClient.Connect(ctx, transport, nil)
    if err != nil {
        panic(err)
    }
    defer session.Close()

	// 2. Discover Tools from MCP Server
	toolsResult, err := session.ListTools(ctx, nil)
	if err != nil {
		log.Fatalf("ListTools Error: %v", err)
	}
	fmt.Printf("Discovered tools: %s\n", jsonStr(toolsResult.Tools))

	// 3. Convert MCP Tools to OpenAI/Ollama Tools
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

	// 4. Configure Ollama Client (OpenAI-compatible)
	config := openai.DefaultConfig("")
	config.BaseURL = "http://host.docker.internal:11434/v1" // Standard Ollama local port
	llm := openai.NewClientWithConfig(config)

	messages := []openai.ChatCompletionMessage{
		{Role: openai.ChatMessageRoleUser, Content: userQuery},
	}

	// 5. Initial Call to LLM
	resp, err := llm.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:    llmModel,
		Messages: messages,
		Tools:    ollamaTools,
	})
	if err != nil {
		log.Fatalf("LLM Error: %v", err)
	}

	msg := resp.Choices[0].Message
	fmt.Printf("LLM Response: %s\n", jsonStr(msg))

	// 6. Handle Tool Calls (The "Wiring")
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
				log.Fatalf("MCP Call Error: %v", err)
			}

			// Capture the tool result (usually text/markdown)
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

		// 7. Final LLM Answer with Tool Data
		finalResp, _ := llm.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
			Model:    llmModel,
			Messages: messages,
		})
		fmt.Println("\n🤖 Response:", finalResp.Choices[0].Message.Content)
	} else {
		fmt.Println("\n🤖 Response:", msg.Content)
	}
}
