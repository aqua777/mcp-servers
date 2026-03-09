package memory

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aqua777/mcp-servers/common"
	"github.com/aqua777/mcp-servers/core/pkg/runtime"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Options holds configuration for the memory server.
type Options struct {
	MemoryFilePath string
}

func init() {
	runtime.Register(common.MCP_Memory, NewServer)
}

func NewServer(ctx context.Context, opts any) (*mcp.Server, error) {
	options, ok := opts.(Options)
	if !ok {
		return nil, fmt.Errorf("expected Options, got %T", opts)
	}

	manager, err := NewKnowledgeGraphManager(options.MemoryFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create knowledge graph manager: %w", err)
	}

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "memory-server",
		Version: "0.6.3",
	}, &mcp.ServerOptions{})

	// create_entities
	server.AddTool(&mcp.Tool{
		Name:        "create_entities",
		Description: "Create multiple new entities in the knowledge graph",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"entities": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"name": map[string]any{
								"type":        "string",
								"description": "The name of the entity",
							},
							"entityType": map[string]any{
								"type":        "string",
								"description": "The type of the entity",
							},
							"observations": map[string]any{
								"type": "array",
								"items": map[string]any{
									"type": "string",
								},
								"description": "An array of observation contents associated with the entity",
							},
						},
						"required": []string{"name", "entityType", "observations"},
					},
				},
			},
			"required": []string{"entities"},
		},
	}, func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args struct {
			Entities []Entity `json:"entities"`
		}
		if err := json.Unmarshal(request.Params.Arguments, &args); err != nil {
			return handleError(err)
		}
		result, err := manager.CreateEntities(args.Entities)
		if err != nil {
			return handleError(err)
		}
		return handleSuccess(result)
	})

	// create_relations
	server.AddTool(&mcp.Tool{
		Name:        "create_relations",
		Description: "Create multiple new relations between entities in the knowledge graph. Relations should be in active voice",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"relations": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"from": map[string]any{
								"type":        "string",
								"description": "The name of the entity where the relation starts",
							},
							"to": map[string]any{
								"type":        "string",
								"description": "The name of the entity where the relation ends",
							},
							"relationType": map[string]any{
								"type":        "string",
								"description": "The type of the relation",
							},
						},
						"required": []string{"from", "to", "relationType"},
					},
				},
			},
			"required": []string{"relations"},
		},
	}, func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args struct {
			Relations []Relation `json:"relations"`
		}
		if err := json.Unmarshal(request.Params.Arguments, &args); err != nil {
			return handleError(err)
		}
		result, err := manager.CreateRelations(args.Relations)
		if err != nil {
			return handleError(err)
		}
		return handleSuccess(result)
	})

	// add_observations
	server.AddTool(&mcp.Tool{
		Name:        "add_observations",
		Description: "Add new observations to existing entities in the knowledge graph",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"observations": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"entityName": map[string]any{
								"type":        "string",
								"description": "The name of the entity to add the observations to",
							},
							"contents": map[string]any{
								"type": "array",
								"items": map[string]any{
									"type": "string",
								},
								"description": "An array of observation contents to add",
							},
						},
						"required": []string{"entityName", "contents"},
					},
				},
			},
			"required": []string{"observations"},
		},
	}, func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args struct {
			Observations []ObservationRequest `json:"observations"`
		}
		if err := json.Unmarshal(request.Params.Arguments, &args); err != nil {
			return handleError(err)
		}
		result, err := manager.AddObservations(args.Observations)
		if err != nil {
			return handleError(err)
		}
		return handleSuccess(result)
	})

	// delete_entities
	server.AddTool(&mcp.Tool{
		Name:        "delete_entities",
		Description: "Delete multiple entities and their associated relations from the knowledge graph",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"entityNames": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "string",
					},
					"description": "An array of entity names to delete",
				},
			},
			"required": []string{"entityNames"},
		},
	}, func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args struct {
			EntityNames []string `json:"entityNames"`
		}
		if err := json.Unmarshal(request.Params.Arguments, &args); err != nil {
			return handleError(err)
		}
		if err := manager.DeleteEntities(args.EntityNames); err != nil {
			return handleError(err)
		}
		return handleSuccessMsg("Entities deleted successfully")
	})

	// delete_observations
	server.AddTool(&mcp.Tool{
		Name:        "delete_observations",
		Description: "Delete specific observations from entities in the knowledge graph",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"deletions": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"entityName": map[string]any{
								"type":        "string",
								"description": "The name of the entity containing the observations",
							},
							"observations": map[string]any{
								"type": "array",
								"items": map[string]any{
									"type": "string",
								},
								"description": "An array of observations to delete",
							},
						},
						"required": []string{"entityName", "observations"},
					},
				},
			},
			"required": []string{"deletions"},
		},
	}, func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args struct {
			Deletions []ObservationDeletion `json:"deletions"`
		}
		if err := json.Unmarshal(request.Params.Arguments, &args); err != nil {
			return handleError(err)
		}
		if err := manager.DeleteObservations(args.Deletions); err != nil {
			return handleError(err)
		}
		return handleSuccessMsg("Observations deleted successfully")
	})

	// delete_relations
	server.AddTool(&mcp.Tool{
		Name:        "delete_relations",
		Description: "Delete multiple relations from the knowledge graph",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"relations": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"from": map[string]any{
								"type": "string",
							},
							"to": map[string]any{
								"type": "string",
							},
							"relationType": map[string]any{
								"type": "string",
							},
						},
						"required": []string{"from", "to", "relationType"},
					},
					"description": "An array of relations to delete",
				},
			},
			"required": []string{"relations"},
		},
	}, func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args struct {
			Relations []Relation `json:"relations"`
		}
		if err := json.Unmarshal(request.Params.Arguments, &args); err != nil {
			return handleError(err)
		}
		if err := manager.DeleteRelations(args.Relations); err != nil {
			return handleError(err)
		}
		return handleSuccessMsg("Relations deleted successfully")
	})

	// read_graph
	server.AddTool(&mcp.Tool{
		Name:        "read_graph",
		Description: "Read the entire knowledge graph",
		InputSchema: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
	}, func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		graph, err := manager.ReadGraph()
		if err != nil {
			return handleError(err)
		}
		return handleSuccess(graph)
	})

	// search_nodes
	server.AddTool(&mcp.Tool{
		Name:        "search_nodes",
		Description: "Search for nodes in the knowledge graph based on a query",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query": map[string]any{
					"type":        "string",
					"description": "The search query to match against entity names, types, and observation content",
				},
			},
			"required": []string{"query"},
		},
	}, func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args struct {
			Query string `json:"query"`
		}
		if err := json.Unmarshal(request.Params.Arguments, &args); err != nil {
			return handleError(err)
		}
		graph, err := manager.SearchNodes(args.Query)
		if err != nil {
			return handleError(err)
		}
		return handleSuccess(graph)
	})

	// open_nodes
	server.AddTool(&mcp.Tool{
		Name:        "open_nodes",
		Description: "Open specific nodes in the knowledge graph by their names",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"names": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "string",
					},
					"description": "An array of entity names to retrieve",
				},
			},
			"required": []string{"names"},
		},
	}, func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args struct {
			Names []string `json:"names"`
		}
		if err := json.Unmarshal(request.Params.Arguments, &args); err != nil {
			return handleError(err)
		}
		graph, err := manager.OpenNodes(args.Names)
		if err != nil {
			return handleError(err)
		}
		return handleSuccess(graph)
	})

	return server, nil
}

func handleSuccess(data interface{}) (*mcp.CallToolResult, error) {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return handleError(err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{

				Text: string(b),
			},
		},
	}, nil
}

func handleSuccessMsg(msg string) (*mcp.CallToolResult, error) {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{

				Text: msg,
			},
		},
	}, nil
}

func handleError(err error) (*mcp.CallToolResult, error) {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{

				Text: fmt.Sprintf("Error: %v", err),
			},
		},
		IsError: true,
	}, nil
}
