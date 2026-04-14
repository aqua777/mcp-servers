# Sequential Thinking MCP Server

An MCP server implementation in Go that provides a tool for dynamic and reflective problem-solving through a structured thinking process.

This is a Go port of the reference implementation.

## Features

- Break down complex problems into manageable steps
- Revise and refine thoughts as understanding deepens
- Branch into alternative paths of reasoning
- Adjust the total number of thoughts dynamically
- Generate and verify solution hypotheses

## Tool

### sequentialthinking

Facilitates a detailed, step-by-step thinking process for problem-solving and analysis.

**Inputs:**
- `thought` (string): The current thinking step
- `nextThoughtNeeded` (boolean): Whether another thought step is needed
- `thoughtNumber` (integer): Current thought number
- `totalThoughts` (integer): Estimated total thoughts needed
- `isRevision` (boolean, optional): Whether this revises previous thinking
- `revisesThought` (integer, optional): Which thought is being reconsidered
- `branchFromThought` (integer, optional): Branching point thought number
- `branchId` (string, optional): Branch identifier
- `needsMoreThoughts` (boolean, optional): If more thoughts are needed

## Usage

The Sequential Thinking tool is designed for:
- Breaking down complex problems into steps
- Planning and design with room for revision
- Analysis that might need course correction
- Problems where the full scope might not be clear initially
- Tasks that need to maintain context over multiple steps
- Situations where irrelevant information needs to be filtered out

To disable logging of thought information to standard error, set the environment variable: `DISABLE_THOUGHT_LOGGING` to `true`.

## Usage

```bash
# Build
go build -o /tmp/sequentialthinking-server ./cmd/sequentialthinking

# Run
/tmp/sequentialthinking-server

# Run with thought logging disabled
DISABLE_THOUGHT_LOGGING=true /tmp/sequentialthinking-server
```

## IDE Configuration

Add to your `mcp_config.json` (Windsurf) or `claude_desktop_config.json` (Claude Desktop / Claude Code):

```json
{
  "mcpServers": {
    "sequentialthinking": {
      "command": "/tmp/sequentialthinking-server",
      "args": []
    }
  }
}
```

To suppress thought logging in the IDE terminal:

```json
{
  "mcpServers": {
    "sequentialthinking": {
      "command": "/tmp/sequentialthinking-server",
      "args": [],
      "env": {
        "DISABLE_THOUGHT_LOGGING": "true"
      }
    }
  }
}
```

## MCP Inspector

```bash
npx @modelcontextprotocol/inspector /tmp/sequentialthinking-server
```

## Examples

See [`../../examples/sequentialthinking`](../../examples/sequentialthinking) for a demo of interacting with this server using the Go MCP SDK.
