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

func runFileSystemMCPServer(args []string) error {
	allowedDirs := krait.GetStringSlice("app.allowed-directories")

	for i, dir := range allowedDirs {
		absDir, err := filepath.Abs(dir)
		if err != nil {
			return fmt.Errorf("error resolving absolute path for %s: %v", dir, err)
		}
		allowedDirs[i] = absDir
	}

	userQuery := strings.Join(args, " ")

	if len(allowedDirs) == 0 {
		return fmt.Errorf("no allowed directories provided. You must specify at least one allowed directory using --allowed-directories or via config")
	}

	if len(userQuery) == 0 {
		return fmt.Errorf("usage: go run main.go --allowed-directories <dir> \"Your query here\"\n\nExample queries:\n  \"List all Go files in the directory\"\n  \"Find all files containing the word TODO\"\n  \"Search for function definitions matching func.*Handler using grep\"\n  \"Show the directory tree\"\n  \"Read the README file\"")
	}

	ctx := context.Background()

	// 1. Setup the actual Transport (the "wire")
	// Start the filesystem MCP server process in AI mode for structured JSON output.
	// We pass the allowed directories as arguments to the command.
	cmdArgs := append([]string{"run", "../../cmd/fs-mcp/main.go", "--ai-mode"}, allowedDirs...)
	execCmd := exec.Command("go", cmdArgs...)
	execCmd.Stderr = os.Stderr

	stdout, err := execCmd.StdoutPipe()
	if err != nil {
		return err
	}
	stdin, err := execCmd.StdinPipe()
	if err != nil {
		return err
	}

	if err := execCmd.Start(); err != nil {
		return err
	}
	defer execCmd.Process.Kill()

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

	systemPrompt := fmt.Sprintf(
		"You are a filesystem assistant. The allowed directories are: %s. "+
			"Use MCP tools to answer questions about files and directories. "+
			"Use the 'grep' tool to search file contents by pattern. "+
			"Use 'search_files' to find files by name/glob. "+
			"Use 'read_text_file', 'list_directory', and 'directory_tree' for reading and browsing.",
		strings.Join(allowedDirs, ", "),
	)

	messages := []openai.ChatCompletionMessage{
		{Role: openai.ChatMessageRoleSystem, Content: systemPrompt},
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
			var toolArgs map[string]interface{}
			json.Unmarshal([]byte(tc.Function.Arguments), &toolArgs)

			callResp, err := session.CallTool(ctx, &mcp.CallToolParams{
				Name:      tc.Function.Name,
				Arguments: toolArgs,
			})
			if err != nil {
				return fmt.Errorf("MCP Call Error: %v", err)
			}

			// Capture the tool result
			toolResultContent, _ := json.Marshal(callResp.Content)
			fmt.Printf("Tool Result:\n---\n%s\n---\n", toolResultContent)

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
	app := krait.App("example-filesystem", "An example chat app that uses the filesystem MCP server", "An example chat app that uses the filesystem MCP server").
		WithConfig("", "config", "c", "APP_CONFIG").
		WithStringSliceP("app.allowed-directories", "List of absolute paths that the MCP server is allowed to access", "allowed-directories", "d", "ALLOWED_DIRECTORIES", []string{}).
		WithRun(runFileSystemMCPServer)

	if err := app.Execute(); err != nil {
		log.Fatal(err)
	}
}
