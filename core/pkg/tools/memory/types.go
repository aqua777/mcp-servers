package memory



// Entity represents a node in the knowledge graph.
type Entity struct {
	Name         string   `json:"name"`
	EntityType   string   `json:"entityType"`
	Observations []string `json:"observations"`
}

// Relation represents a directed edge between two entities in the knowledge graph.
type Relation struct {
	From         string `json:"from"`
	To           string `json:"to"`
	RelationType string `json:"relationType"`
}

// KnowledgeGraph represents the entire graph containing entities and relations.
type KnowledgeGraph struct {
	Entities  []Entity   `json:"entities"`
	Relations []Relation `json:"relations"`
}

// ObservationRequest represents a request to add observations to an entity.
type ObservationRequest struct {
	EntityName string   `json:"entityName"`
	Contents   []string `json:"contents"`
}

// ObservationResult represents the result of adding observations.
type ObservationResult struct {
	EntityName        string   `json:"entityName"`
	AddedObservations []string `json:"addedObservations"`
}

// ObservationDeletion represents a request to delete specific observations from an entity.
type ObservationDeletion struct {
	EntityName   string   `json:"entityName"`
	Observations []string `json:"observations"`
}

// lineItem is used for JSONL serialization/deserialization.
type lineItem struct {
	Type         string   `json:"type"`
	Name         string   `json:"name,omitempty"`
	EntityType   string   `json:"entityType,omitempty"`
	Observations []string `json:"observations,omitempty"`
	From         string   `json:"from,omitempty"`
	To           string   `json:"to,omitempty"`
	RelationType string   `json:"relationType,omitempty"`
}
