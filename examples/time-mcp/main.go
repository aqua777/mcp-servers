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
	llmModel = "qwen3:0.6b"
)

func runTimeMCPServer(args []string) error {
	userQuery := strings.Join(args, " ")

	if len(userQuery) == 0 {
		return fmt.Errorf("usage: go run main.go [flags] \"Your query here, e.g. What is the current time in Europe/Paris?\"")
	}

	ctx := context.Background()

	cmd := exec.Command("go", "run", "../../cmd/time-mcp/main.go")
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

	mcpClient := mcp.NewClient(&mcp.Implementation{
		Name:    "my-time-client",
		Version: "1.0.0",
	}, nil)

	session, err := mcpClient.Connect(ctx, transport, nil)
	if err != nil {
		return err
	}
	defer session.Close()

	toolsResult, err := session.ListTools(ctx, nil)
	if err != nil {
		return fmt.Errorf("ListTools Error: %v", err)
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
		return fmt.Errorf("LLM Error: %v", err)
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
				return fmt.Errorf("MCP Call Error: %v", err)
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
	return nil
}

func main() {
	app := krait.App("example-time", "An example chat app that uses the time MCP server", "An example chat app that uses the time MCP server").
		WithConfig("", "config", "c", "APP_CONFIG").
		WithRun(runTimeMCPServer)

	if err := app.Execute(); err != nil {
		log.Fatal(err)
	}
}
