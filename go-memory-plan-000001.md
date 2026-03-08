# Memory Server Golang Implementation Plan

## Overview
This document outlines the plan for implementing the `memory` MCP server in Golang, mirroring the functionality of the TypeScript reference implementation found in `servers/src/memory`. The memory server provides a simple knowledge graph structure with entities, relations, and observations, stored in a JSONL file.

## TypeScript Reference Analysis
The TypeScript implementation (`servers/src/memory/index.ts`) provides a Knowledge Graph Manager with the following features:
- File-based storage (`memory.jsonl` with fallback/migration from `memory.json`)
- Entities (name, entityType, observations array)
- Relations (from, to, relationType)
- Tools exposed via MCP:
  - `create_entities`
  - `create_relations`
  - `add_observations`
  - `delete_entities`
  - `delete_observations`
  - `delete_relations`
  - `read_graph`
  - `search_nodes`
  - `open_nodes`

## Proposed Golang Structure

Following the established pattern in the `golang` directory and the rules in `AGENTS.md`:

### 1. New Module
- Create a new module in `golang/cmd/memory` and add it to `golang/go.work`.

### 2. Command Entry Point & Core Logic
`golang/cmd/memory/main.go`
- Initialization of the MCP server
- Registration of tools
- Environment variable handling
- Starting the stdio server

`golang/cmd/memory/manager.go`
- `KnowledgeGraphManager` implementation
- File handling, JSONL parsing, migration logic

### 3. Examples
- Create `golang/examples/memory` directory demonstrating how to use the server.

### 4. Documentation
- Create `golang/cmd/memory/README.md` with usage instructions.
- Update `golang/docs/parity.md` with a section comparing the Go implementation to the reference TypeScript implementation.

### 5. Testing
- Create `golang/cmd/memory/manager_test.go` using `github.com/stretchr/testify/suite`.
- Ensure minimum 90% coverage.
- Document any exceptions that are difficult to simulate.

## Implementation Steps

### Phase 1: Setup and Types
1. Initialize new Go module in `golang/cmd/memory` and add it to `golang/go.work`.
2. Define the data structures for Entities, Relations, and KnowledgeGraph.

### Phase 2: Core Logic and Testing
1. Implement file reading/writing (JSONL format handling) and graph operations (CRUD) in `manager.go`.
2. Implement memory file migration (`memory.json` to `memory.jsonl`) logic.
3. Write comprehensive unit tests (`manager_test.go`) using `testify/suite` to achieve >90% coverage.

### Phase 3: Create MCP Tool Wrappers and Command Line App
1. Create `golang/cmd/memory/main.go`.
2. Wrap the `KnowledgeGraphManager` methods into MCP-compatible tool definitions and handler functions.
3. Setup `stdio` transport and start the server.

### Phase 4: Examples and Documentation
1. Create example usage in `golang/examples/memory`.
2. Add a `README.md` in `golang/cmd/memory`.
3. Update `golang/docs/parity.md` detailing implementation parity.

## Tool Signatures (Golang)

*   `create_entities(entities []Entity) []Entity`
*   `create_relations(relations []Relation) []Relation`
*   `add_observations(observations []ObservationRequest) []ObservationResult`
*   `delete_entities(entityNames []string)`
*   `delete_observations(deletions []ObservationDeletion)`
*   `delete_relations(relations []Relation)`
*   `read_graph() KnowledgeGraph`
*   `search_nodes(query string) KnowledgeGraph`
*   `open_nodes(names []string) KnowledgeGraph`
