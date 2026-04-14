package sequentialthinking

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/aqua777/mcp-servers/common"
	"github.com/aqua777/mcp-servers/core/pkg/runtime"
	"github.com/mattn/go-runewidth"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func init() {
	runtime.Register(common.MCP_SequentialThinking, NewServer)
}

// ThoughtData represents a single thought step
type ThoughtData struct {
	Thought           string  `json:"thought"`
	ThoughtNumber     int     `json:"thoughtNumber"`
	TotalThoughts     int     `json:"totalThoughts"`
	IsRevision        *bool   `json:"isRevision,omitempty"`
	RevisesThought    *int    `json:"revisesThought,omitempty"`
	BranchFromThought *int    `json:"branchFromThought,omitempty"`
	BranchID          *string `json:"branchId,omitempty"`
	NeedsMoreThoughts *bool   `json:"needsMoreThoughts,omitempty"`
	NextThoughtNeeded bool    `json:"nextThoughtNeeded"`
}

type ServerState struct {
	thoughtHistory        []ThoughtData
	branches              map[string][]ThoughtData
	disableThoughtLogging bool
}

func NewServerState() *ServerState {
	return &ServerState{
		thoughtHistory:        make([]ThoughtData, 0),
		branches:              make(map[string][]ThoughtData),
		disableThoughtLogging: strings.ToLower(os.Getenv("DISABLE_THOUGHT_LOGGING")) == "true",
	}
}

func (s *ServerState) formatThought(thoughtData ThoughtData) string {
	var prefix string
	var contextStr string

	if thoughtData.IsRevision != nil && *thoughtData.IsRevision {
		prefix = "\x1b[33m🔄 Revision\x1b[0m" // Yellow
		if thoughtData.RevisesThought != nil {
			contextStr = fmt.Sprintf(" (revising thought %d)", *thoughtData.RevisesThought)
		}
	} else if thoughtData.BranchFromThought != nil && thoughtData.BranchID != nil {
		prefix = "\x1b[32m🌿 Branch\x1b[0m" // Green
		contextStr = fmt.Sprintf(" (from thought %d, ID: %s)", *thoughtData.BranchFromThought, *thoughtData.BranchID)
	} else {
		prefix = "\x1b[34m💭 Thought\x1b[0m" // Blue
	}

	header := fmt.Sprintf("%s %d/%d%s", prefix, thoughtData.ThoughtNumber, thoughtData.TotalThoughts, contextStr)

	cleanHeader := stripANSI(header)

	thoughtLines := strings.Split(thoughtData.Thought, "\n")
	maxThoughtLen := 0
	for _, line := range thoughtLines {
		lineWidth := runewidth.StringWidth(line)
		if lineWidth > maxThoughtLen {
			maxThoughtLen = lineWidth
		}
	}

	headerWidth := runewidth.StringWidth(cleanHeader)
	borderLen := max(headerWidth, maxThoughtLen) + 4
	border := strings.Repeat("─", borderLen)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("\n┌%s┐\n", border))
	sb.WriteString(fmt.Sprintf("│ %s%s │\n", header, strings.Repeat(" ", borderLen-headerWidth-2)))
	sb.WriteString(fmt.Sprintf("├%s┤\n", border))

	for _, line := range thoughtLines {
		lineWidth := runewidth.StringWidth(line)
		sb.WriteString(fmt.Sprintf("│ %s%s │\n", line, strings.Repeat(" ", borderLen-lineWidth-2)))
	}
	sb.WriteString(fmt.Sprintf("└%s┘", border))

	return sb.String()
}

func stripANSI(str string) string {
	str = strings.ReplaceAll(str, "\x1b[33m", "")
	str = strings.ReplaceAll(str, "\x1b[32m", "")
	str = strings.ReplaceAll(str, "\x1b[34m", "")
	str = strings.ReplaceAll(str, "\x1b[0m", "")
	return str
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

type ProcessResult struct {
	ThoughtNumber        int      `json:"thoughtNumber"`
	TotalThoughts        int      `json:"totalThoughts"`
	NextThoughtNeeded    bool     `json:"nextThoughtNeeded"`
	Branches             []string `json:"branches"`
	ThoughtHistoryLength int      `json:"thoughtHistoryLength"`
}

func handleCallTool(ctx context.Context, request *mcp.CallToolRequest, state *ServerState) (*mcp.CallToolResult, error) {
	var input ThoughtData
	if err := json.Unmarshal(request.Params.Arguments, &input); err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{
				Text: fmt.Sprintf(`{"error": "Invalid arguments: %v", "status": "failed"}`, err),
			}},
			IsError: true,
		}, nil
	}

	if input.ThoughtNumber > input.TotalThoughts {
		input.TotalThoughts = input.ThoughtNumber
	}

	state.thoughtHistory = append(state.thoughtHistory, input)

	if input.BranchFromThought != nil && input.BranchID != nil {
		if state.branches[*input.BranchID] == nil {
			state.branches[*input.BranchID] = make([]ThoughtData, 0)
		}
		state.branches[*input.BranchID] = append(state.branches[*input.BranchID], input)
	}

	if !state.disableThoughtLogging {
		fmt.Fprintln(os.Stderr, state.formatThought(input))
	}

	branches := make([]string, 0, len(state.branches))
	for k := range state.branches {
		branches = append(branches, k)
	}

	res := ProcessResult{
		ThoughtNumber:        input.ThoughtNumber,
		TotalThoughts:        input.TotalThoughts,
		NextThoughtNeeded:    input.NextThoughtNeeded,
		Branches:             branches,
		ThoughtHistoryLength: len(state.thoughtHistory),
	}

	resBytes, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		errorJSON := fmt.Sprintf(`{"error": "%s", "status": "failed"}`, err.Error())
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: errorJSON}},
			IsError: true,
		}, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(resBytes)}},
	}, nil
}

func NewServer(ctx context.Context, opts any) (*mcp.Server, error) {
	state := NewServerState()

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "sequential-thinking-server",
		Version: "0.2.0",
	}, &mcp.ServerOptions{})

	inputSchema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"thought": map[string]any{
				"type":        "string",
				"description": "Your current thinking step",
			},
			"nextThoughtNeeded": map[string]any{
				"type":        "boolean",
				"description": "Whether another thought step is needed",
			},
			"thoughtNumber": map[string]any{
				"type":        "integer",
				"description": "Current thought number (numeric value, e.g., 1, 2, 3)",
				"minimum":     1,
			},
			"totalThoughts": map[string]any{
				"type":        "integer",
				"description": "Estimated total thoughts needed (numeric value, e.g., 5, 10)",
				"minimum":     1,
			},
			"isRevision": map[string]any{
				"type":        "boolean",
				"description": "Whether this revises previous thinking",
			},
			"revisesThought": map[string]any{
				"type":        "integer",
				"description": "Which thought is being reconsidered",
				"minimum":     1,
			},
			"branchFromThought": map[string]any{
				"type":        "integer",
				"description": "Branching point thought number",
				"minimum":     1,
			},
			"branchId": map[string]any{
				"type":        "string",
				"description": "Branch identifier",
			},
			"needsMoreThoughts": map[string]any{
				"type":        "boolean",
				"description": "If more thoughts are needed",
			},
		},
		"required": []string{"thought", "nextThoughtNeeded", "thoughtNumber", "totalThoughts"},
	}

	server.AddTool(&mcp.Tool{
		Name: common.MCP_SequentialThinking,
		Description: `A detailed tool for dynamic and reflective problem-solving through thoughts.
This tool helps analyze problems through a flexible thinking process that can adapt and evolve.
Each thought can build on, question, or revise previous insights as understanding deepens.

When to use this tool:
- Breaking down complex problems into steps
- Planning and design with room for revision
- Analysis that might need course correction
- Problems where the full scope might not be clear initially
- Problems that require a multi-step solution
- Tasks that need to maintain context over multiple steps
- Situations where irrelevant information needs to be filtered out

Key features:
- You can adjust totalThoughts up or down as you progress
- You can question or revise previous thoughts
- You can add more thoughts even after reaching what seemed like the end
- You can express uncertainty and explore alternative approaches
- Not every thought needs to build linearly - you can branch or backtrack
- Generates a solution hypothesis
- Verifies the hypothesis based on the Chain of Thought steps
- Repeats the process until satisfied
- Provides a correct answer

Parameters explained:
- thought: Your current thinking step, which can include:
  * Regular analytical steps
  * Revisions of previous thoughts
  * Questions about previous decisions
  * Realizations about needing more analysis
  * Changes in approach
  * Hypothesis generation
  * Hypothesis verification
- nextThoughtNeeded: True if you need more thinking, even if at what seemed like the end
- thoughtNumber: Current number in sequence (can go beyond initial total if needed)
- totalThoughts: Current estimate of thoughts needed (can be adjusted up/down)
- isRevision: A boolean indicating if this thought revises previous thinking
- revisesThought: If isRevision is true, which thought number is being reconsidered
- branchFromThought: If branching, which thought number is the branching point
- branchId: Identifier for the current branch (if any)
- needsMoreThoughts: If reaching end but realizing more thoughts needed

You should:
1. Start with an initial estimate of needed thoughts, but be ready to adjust
2. Feel free to question or revise previous thoughts
3. Don't hesitate to add more thoughts if needed, even at the "end"
4. Express uncertainty when present
5. Mark thoughts that revise previous thinking or branch into new paths
6. Ignore information that is irrelevant to the current step
7. Generate a solution hypothesis when appropriate
8. Verify the hypothesis based on the Chain of Thought steps
9. Repeat the process until satisfied with the solution
10. Provide a single, ideally correct answer as the final output
11. Only set nextThoughtNeeded to false when truly done and a satisfactory answer is reached`,
		InputSchema: inputSchema,
	}, func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleCallTool(ctx, request, state)
	})

	return server, nil
}
