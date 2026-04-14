package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/aqua777/mcp-servers/examples/utils"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/sashabaranov/go-openai"
)

const (
	llmModel = "qwen3:0.6b"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: go run main.go \"Your query here, e.g. What is the current time in Europe/Paris?\"")
	}
	userQuery := strings.Join(os.Args[1:], " ")
	ctx := context.Background()

	cmd := exec.Command("go", "run", "../../cmd/time/main.go")
	cmd.Stderr = os.Stderr

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		log.Fatal(err)
	}

	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}
	defer cmd.Process.Kill()

	transport := &mcp.IOTransport{
		Reader: stdout,
		Writer: stdin,
	}

	mcpClient := mcp.NewClient(&mcp.Implementation{
		Name:    "my-time-client",
		Version: "1.0.0",
	}, nil)

	session, err := mcpClient.Connect(ctx, transport, nil)
	if err != nil {
		panic(err)
	}
	defer session.Close()

	toolsResult, err := session.ListTools(ctx, nil)
	if err != nil {
		log.Fatalf("ListTools Error: %v", err)
	}
	fmt.Printf("Discovered tools: %s\n", utils.JsonStr(toolsResult.Tools))

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

	config := openai.DefaultConfig("")
	config.BaseURL = "http://host.docker.internal:11434/v1"
	llm := openai.NewClientWithConfig(config)

	messages := []openai.ChatCompletionMessage{
		{Role: openai.ChatMessageRoleUser, Content: userQuery},
	}

	resp, err := llm.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:    llmModel,
		Messages: messages,
		Tools:    ollamaTools,
	})
	if err != nil {
		log.Fatalf("LLM Error: %v", err)
	}

	msg := resp.Choices[0].Message
	fmt.Printf("LLM Response: %s\n", utils.JsonStr(msg))

	if len(msg.ToolCalls) > 0 {
		for _, tc := range msg.ToolCalls {
			fmt.Printf("🔧 LLM calling tool: %s\n", tc.Function.Name)

			var args map[string]interface{}
			json.Unmarshal([]byte(tc.Function.Arguments), &args)

			callResp, err := session.CallTool(ctx, &mcp.CallToolParams{
				Name:      tc.Function.Name,
				Arguments: args,
			})
			if err != nil {
				log.Fatalf("MCP Call Error: %v", err)
			}

			toolResultContent, _ := json.Marshal(callResp.Content)
			fmt.Printf("Tool Result:\n---\n%s\n---\n", toolResultContent)

			messages = append(messages, msg)
			messages = append(messages, openai.ChatCompletionMessage{
				Role:       openai.ChatMessageRoleTool,
				Content:    string(toolResultContent),
				ToolCallID: tc.ID,
			})
		}

		finalResp, _ := llm.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
			Model:    llmModel,
			Messages: messages,
		})
		fmt.Println("\n🤖 Response:", finalResp.Choices[0].Message.Content)
	} else {
		fmt.Println("\n🤖 Response:", msg.Content)
	}
}
