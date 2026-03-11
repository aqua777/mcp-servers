# Memory MCP Server (Golang)

This is a Golang implementation of the Memory MCP Server, designed to mirror the reference TypeScript implementation. It provides a simple knowledge graph structure with entities, relations, and observations, stored in a JSONL file.

## Features
- File-based storage using `.jsonl` (with automatic migration from legacy `.json` files)
- Full CRUD operations for Entities, Relations, and Observations

## Configuration
You can specify the memory file path using the `krait` flags or environment variables:
- **Flag**: `--memory-file-path` or `-m` (e.g. `./memory --memory-file-path /path/to/memory.jsonl`)
- **Environment Variable**: `MEMORY_FILE_PATH`
- **Default**: If neither is provided, it defaults to `memory.jsonl` in the executable's directory.

## Tools Exposed
- `create_entities`: Create multiple new entities.
- `create_relations`: Create multiple new relations between entities.
- `add_observations`: Add new observations to existing entities.
- `delete_entities`: Delete multiple entities and their associated relations.
- `delete_observations`: Delete specific observations from entities.
- `delete_relations`: Delete multiple relations from the knowledge graph.
- `read_graph`: Read the entire knowledge graph.
- `search_nodes`: Search for nodes based on a text query.
- `open_nodes`: Open specific nodes by their names.

## Examples
See `../../examples/memory` for an example of how to interact with this server using the Go SDK.
