package sequentialthinking

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/aqua777/mcp-servers/common"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/suite"
)

type SequentialThinkingTestSuite struct {
	suite.Suite
	state *ServerState
}

func (s *SequentialThinkingTestSuite) SetupTest() {
	s.state = NewServerState()
}

func TestSequentialThinkingTestSuite(t *testing.T) {
	suite.Run(t, new(SequentialThinkingTestSuite))
}

func ptr[T any](v T) *T {
	return &v
}

func (s *SequentialThinkingTestSuite) TestStripANSI() {
	s.Equal("hello", stripANSI("\x1b[33mhello\x1b[0m"))
	s.Equal("world", stripANSI("\x1b[32mworld\x1b[0m"))
	s.Equal("test", stripANSI("\x1b[34mtest\x1b[0m"))
}

func (s *SequentialThinkingTestSuite) TestMax() {
	s.Equal(5, max(3, 5))
	s.Equal(5, max(5, 3))
	s.Equal(5, max(5, 5))
}

func (s *SequentialThinkingTestSuite) TestFormatThought() {
	// Base thought
	thoughtData := ThoughtData{
		Thought:           "Initial thought",
		ThoughtNumber:     1,
		TotalThoughts:     5,
		NextThoughtNeeded: true,
	}

	formatted := s.state.formatThought(thoughtData)
	s.True(strings.Contains(formatted, "💭 Thought"))
	s.True(strings.Contains(formatted, "1/5"))
	s.True(strings.Contains(formatted, "Initial thought"))

	// Revision thought
	thoughtData = ThoughtData{
		Thought:           "Revision thought",
		ThoughtNumber:     2,
		TotalThoughts:     5,
		IsRevision:        ptr(true),
		RevisesThought:    ptr(1),
		NextThoughtNeeded: true,
	}

	formatted = s.state.formatThought(thoughtData)
	s.True(strings.Contains(formatted, "🔄 Revision"))
	s.True(strings.Contains(formatted, "2/5"))
	s.True(strings.Contains(formatted, "revising thought 1"))
	s.True(strings.Contains(formatted, "Revision thought"))

	// Branch thought
	thoughtData = ThoughtData{
		Thought:           "Branch thought",
		ThoughtNumber:     3,
		TotalThoughts:     5,
		BranchFromThought: ptr(2),
		BranchID:          ptr("b1"),
		NextThoughtNeeded: true,
	}

	formatted = s.state.formatThought(thoughtData)
	s.True(strings.Contains(formatted, "🌿 Branch"))
	s.True(strings.Contains(formatted, "3/5"))
	s.True(strings.Contains(formatted, "from thought 2, ID: b1"))
	s.True(strings.Contains(formatted, "Branch thought"))

	// Multi-line thought
	thoughtData = ThoughtData{
		Thought:           "Multi-line\nthought\nhere",
		ThoughtNumber:     4,
		TotalThoughts:     5,
		NextThoughtNeeded: true,
	}

	formatted = s.state.formatThought(thoughtData)
	s.True(strings.Contains(formatted, "Multi-line"))
	s.True(strings.Contains(formatted, "thought"))
	s.True(strings.Contains(formatted, "here"))
}

func (s *SequentialThinkingTestSuite) TestProcessThoughtCoreLogic() {
	os.Setenv("DISABLE_THOUGHT_LOGGING", "true")
	s.state = NewServerState()

	// Adjust totalThoughts if thoughtNumber > totalThoughts
	input2 := ThoughtData{
		Thought:           "Second thought",
		ThoughtNumber:     4,
		TotalThoughts:     3,
		NextThoughtNeeded: true,
	}

	if input2.ThoughtNumber > input2.TotalThoughts {
		input2.TotalThoughts = input2.ThoughtNumber
	}
	s.Equal(4, input2.TotalThoughts)

	// Branch logic
	input3 := ThoughtData{
		Thought:           "Branch thought",
		ThoughtNumber:     2,
		TotalThoughts:     4,
		BranchFromThought: ptr(1),
		BranchID:          ptr("test-branch"),
		NextThoughtNeeded: true,
	}

	s.state.thoughtHistory = append(s.state.thoughtHistory, input3)
	if input3.BranchFromThought != nil && input3.BranchID != nil {
		if s.state.branches[*input3.BranchID] == nil {
			s.state.branches[*input3.BranchID] = make([]ThoughtData, 0)
		}
		s.state.branches[*input3.BranchID] = append(s.state.branches[*input3.BranchID], input3)
	}

	s.Len(s.state.branches["test-branch"], 1)
	s.Equal("Branch thought", s.state.branches["test-branch"][0].Thought)
}

func (s *SequentialThinkingTestSuite) TestProcessResultMarshal() {
	res := ProcessResult{
		ThoughtNumber:        1,
		TotalThoughts:        5,
		NextThoughtNeeded:    true,
		Branches:             []string{"branch1", "branch2"},
		ThoughtHistoryLength: 3,
	}

	resBytes, err := json.MarshalIndent(res, "", "  ")
	s.NoError(err)
	s.True(strings.Contains(string(resBytes), `"thoughtNumber": 1`))
	s.True(strings.Contains(string(resBytes), `"branches": [`))
}

func (s *SequentialThinkingTestSuite) TestNewServer() {
	server, err := NewServer(context.Background(), nil)
	s.NoError(err)
	s.NotNil(server)
}

func (s *SequentialThinkingTestSuite) TestHandleCallTool() {
	state := NewServerState()

	// Valid request
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      common.MCP_SequentialThinking,
			Arguments: []byte(`{"thought": "test", "nextThoughtNeeded": false, "thoughtNumber": 1, "totalThoughts": 1}`),
		},
	}

	result, err := handleCallTool(context.Background(), req, state)
	s.NoError(err)
	s.False(result.IsError)
	s.Contains(result.Content[0].(*mcp.TextContent).Text, `"thoughtNumber": 1`)

	// Invalid JSON request
	reqInvalid := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      common.MCP_SequentialThinking,
			Arguments: []byte(`{"thought": "test", "nextThoughtNeeded": "invalid type"}`),
		},
	}

	resultInvalid, err := handleCallTool(context.Background(), reqInvalid, state)
	s.NoError(err)
	s.True(resultInvalid.IsError)
	s.Contains(resultInvalid.Content[0].(*mcp.TextContent).Text, `Invalid arguments`)

	// Test with logging enabled
	os.Setenv("DISABLE_THOUGHT_LOGGING", "false")
	stateLogging := NewServerState()
	resultLogging, err := handleCallTool(context.Background(), req, stateLogging)
	s.NoError(err)
	s.False(resultLogging.IsError)
}

func (s *SequentialThinkingTestSuite) TestHandleCallToolBranchLogic() {
	state := NewServerState()

	// Test branch creation
	reqBranch := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      common.MCP_SequentialThinking,
			Arguments: []byte(`{"thought": "branch test", "nextThoughtNeeded": false, "thoughtNumber": 2, "totalThoughts": 2, "branchFromThought": 1, "branchId": "b1"}`),
		},
	}

	result, err := handleCallTool(context.Background(), reqBranch, state)
	s.NoError(err)
	s.False(result.IsError)
	s.Contains(result.Content[0].(*mcp.TextContent).Text, `"branches": [`)
	s.Len(state.branches, 1)
	s.Len(state.branches["b1"], 1)
}

func (s *SequentialThinkingTestSuite) TestHandleCallToolAdjustTotalThoughts() {
	state := NewServerState()

	// Test auto-adjustment of totalThoughts when thoughtNumber exceeds it
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      common.MCP_SequentialThinking,
			Arguments: []byte(`{"thought": "test", "nextThoughtNeeded": false, "thoughtNumber": 5, "totalThoughts": 3}`),
		},
	}

	result, err := handleCallTool(context.Background(), req, state)
	s.NoError(err)
	s.False(result.IsError)
	s.Contains(result.Content[0].(*mcp.TextContent).Text, `"totalThoughts": 5`)
}
