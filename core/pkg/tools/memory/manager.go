package memory

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// KnowledgeGraphManager handles interactions with the knowledge graph.
type KnowledgeGraphManager struct {
	memoryFilePath string
}

// NewKnowledgeGraphManager creates a new instance and ensures the memory file path is initialized.
func NewKnowledgeGraphManager(memoryFilePath string) (*KnowledgeGraphManager, error) {
	// Let's implement migration from old memory.json to memory.jsonl if needed
	if memoryFilePath == "" {
		// Use default
		ex, err := os.Executable()
		if err != nil {
			return nil, err
		}
		exPath := filepath.Dir(ex)
		memoryFilePath = filepath.Join(exPath, "memory.jsonl")
	}

	// For backward compatibility: if a custom path is provided but it doesn't end in .jsonl,
	// or we just want to ensure we are using .jsonl, we handle migration.
	// But in Go we'll just stick to what was given, or check if the old file exists.
	dir := filepath.Dir(memoryFilePath)
	base := filepath.Base(memoryFilePath)
	
	if base == "memory.jsonl" {
		oldMemoryPath := filepath.Join(dir, "memory.json")
		
		// Check if old file exists and new file doesn't
		if _, err := os.Stat(oldMemoryPath); err == nil {
			if _, err := os.Stat(memoryFilePath); os.IsNotExist(err) {
				// Migrate
				err = os.Rename(oldMemoryPath, memoryFilePath)
				if err != nil {
					return nil, err
				}
			}
		}
	}

	return &KnowledgeGraphManager{
		memoryFilePath: memoryFilePath,
	}, nil
}

// loadGraph loads the knowledge graph from the memory file.
func (m *KnowledgeGraphManager) loadGraph() (*KnowledgeGraph, error) {
	graph := &KnowledgeGraph{
		Entities:  []Entity{},
		Relations: []Relation{},
	}

	file, err := os.Open(m.memoryFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return graph, nil
		}
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var item lineItem
		if err := json.Unmarshal([]byte(line), &item); err != nil {
			continue // Skip malformed lines
		}

		if item.Type == "entity" {
			graph.Entities = append(graph.Entities, Entity{
				Name:         item.Name,
				EntityType:   item.EntityType,
				Observations: item.Observations,
			})
		} else if item.Type == "relation" {
			graph.Relations = append(graph.Relations, Relation{
				From:         item.From,
				To:           item.To,
				RelationType: item.RelationType,
			})
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return graph, nil
}

// saveGraph saves the knowledge graph to the memory file.
func (m *KnowledgeGraphManager) saveGraph(graph *KnowledgeGraph) error {
	file, err := os.Create(m.memoryFilePath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)

	for _, e := range graph.Entities {
		item := lineItem{
			Type:         "entity",
			Name:         e.Name,
			EntityType:   e.EntityType,
			Observations: e.Observations,
		}
		data, err := json.Marshal(item)
		if err != nil {
			continue
		}
		writer.Write(data)
		writer.WriteByte('\n')
	}

	for _, r := range graph.Relations {
		item := lineItem{
			Type:         "relation",
			From:         r.From,
			To:           r.To,
			RelationType: r.RelationType,
		}
		data, err := json.Marshal(item)
		if err != nil {
			continue
		}
		writer.Write(data)
		writer.WriteByte('\n')
	}

	return writer.Flush()
}

// CreateEntities creates new entities in the knowledge graph.
func (m *KnowledgeGraphManager) CreateEntities(entities []Entity) ([]Entity, error) {
	graph, err := m.loadGraph()
	if err != nil {
		return nil, err
	}

	var newEntities []Entity
	for _, e := range entities {
		exists := false
		for _, existingEntity := range graph.Entities {
			if existingEntity.Name == e.Name {
				exists = true
				break
			}
		}
		if !exists {
			newEntities = append(newEntities, e)
		}
	}

	graph.Entities = append(graph.Entities, newEntities...)
	if err := m.saveGraph(graph); err != nil {
		return nil, err
	}

	return newEntities, nil
}

// CreateRelations creates new relations in the knowledge graph.
func (m *KnowledgeGraphManager) CreateRelations(relations []Relation) ([]Relation, error) {
	graph, err := m.loadGraph()
	if err != nil {
		return nil, err
	}

	var newRelations []Relation
	for _, r := range relations {
		exists := false
		for _, existingRelation := range graph.Relations {
			if existingRelation.From == r.From &&
				existingRelation.To == r.To &&
				existingRelation.RelationType == r.RelationType {
				exists = true
				break
			}
		}
		if !exists {
			newRelations = append(newRelations, r)
		}
	}

	graph.Relations = append(graph.Relations, newRelations...)
	if err := m.saveGraph(graph); err != nil {
		return nil, err
	}

	return newRelations, nil
}

// AddObservations adds observations to existing entities.
func (m *KnowledgeGraphManager) AddObservations(observations []ObservationRequest) ([]ObservationResult, error) {
	graph, err := m.loadGraph()
	if err != nil {
		return nil, err
	}

	var results []ObservationResult
	for _, req := range observations {
		var targetEntity *Entity
		for i := range graph.Entities {
			if graph.Entities[i].Name == req.EntityName {
				targetEntity = &graph.Entities[i]
				break
			}
		}

		if targetEntity == nil {
			return nil, errors.New("Entity with name " + req.EntityName + " not found")
		}

		var newObservations []string
		for _, obs := range req.Contents {
			exists := false
			for _, existingObs := range targetEntity.Observations {
				if existingObs == obs {
					exists = true
					break
				}
			}
			if !exists {
				newObservations = append(newObservations, obs)
			}
		}

		targetEntity.Observations = append(targetEntity.Observations, newObservations...)
		results = append(results, ObservationResult{
			EntityName:        req.EntityName,
			AddedObservations: newObservations,
		})
	}

	if err := m.saveGraph(graph); err != nil {
		return nil, err
	}

	return results, nil
}

// DeleteEntities deletes entities and their associated relations.
func (m *KnowledgeGraphManager) DeleteEntities(entityNames []string) error {
	graph, err := m.loadGraph()
	if err != nil {
		return err
	}

	namesToDelete := make(map[string]bool)
	for _, name := range entityNames {
		namesToDelete[name] = true
	}

	var newEntities []Entity
	for _, e := range graph.Entities {
		if !namesToDelete[e.Name] {
			newEntities = append(newEntities, e)
		}
	}
	graph.Entities = newEntities

	var newRelations []Relation
	for _, r := range graph.Relations {
		if !namesToDelete[r.From] && !namesToDelete[r.To] {
			newRelations = append(newRelations, r)
		}
	}
	graph.Relations = newRelations

	return m.saveGraph(graph)
}

// DeleteObservations deletes specific observations from entities.
func (m *KnowledgeGraphManager) DeleteObservations(deletions []ObservationDeletion) error {
	graph, err := m.loadGraph()
	if err != nil {
		return err
	}

	for _, del := range deletions {
		obsToDelete := make(map[string]bool)
		for _, obs := range del.Observations {
			obsToDelete[obs] = true
		}

		for i := range graph.Entities {
			if graph.Entities[i].Name == del.EntityName {
				var newObs []string
				for _, obs := range graph.Entities[i].Observations {
					if !obsToDelete[obs] {
						newObs = append(newObs, obs)
					}
				}
				graph.Entities[i].Observations = newObs
				break
			}
		}
	}

	return m.saveGraph(graph)
}

// DeleteRelations deletes specific relations from the graph.
func (m *KnowledgeGraphManager) DeleteRelations(relations []Relation) error {
	graph, err := m.loadGraph()
	if err != nil {
		return err
	}

	var newRelations []Relation
	for _, r := range graph.Relations {
		shouldDelete := false
		for _, del := range relations {
			if r.From == del.From && r.To == del.To && r.RelationType == del.RelationType {
				shouldDelete = true
				break
			}
		}
		if !shouldDelete {
			newRelations = append(newRelations, r)
		}
	}
	graph.Relations = newRelations

	return m.saveGraph(graph)
}

// ReadGraph returns the entire knowledge graph.
func (m *KnowledgeGraphManager) ReadGraph() (*KnowledgeGraph, error) {
	return m.loadGraph()
}

// SearchNodes searches for nodes based on a query.
func (m *KnowledgeGraphManager) SearchNodes(query string) (*KnowledgeGraph, error) {
	graph, err := m.loadGraph()
	if err != nil {
		return nil, err
	}

	queryLower := strings.ToLower(query)

	var filteredEntities []Entity
	filteredEntityNames := make(map[string]bool)

	for _, e := range graph.Entities {
		match := strings.Contains(strings.ToLower(e.Name), queryLower) ||
			strings.Contains(strings.ToLower(e.EntityType), queryLower)

		if !match {
			for _, obs := range e.Observations {
				if strings.Contains(strings.ToLower(obs), queryLower) {
					match = true
					break
				}
			}
		}

		if match {
			filteredEntities = append(filteredEntities, e)
			filteredEntityNames[e.Name] = true
		}
	}

	var filteredRelations []Relation
	for _, r := range graph.Relations {
		if filteredEntityNames[r.From] && filteredEntityNames[r.To] {
			filteredRelations = append(filteredRelations, r)
		}
	}

	return &KnowledgeGraph{
		Entities:  filteredEntities,
		Relations: filteredRelations,
	}, nil
}

// OpenNodes retrieves specific nodes by their names.
func (m *KnowledgeGraphManager) OpenNodes(names []string) (*KnowledgeGraph, error) {
	graph, err := m.loadGraph()
	if err != nil {
		return nil, err
	}

	nameMap := make(map[string]bool)
	for _, name := range names {
		nameMap[name] = true
	}

	var filteredEntities []Entity
	filteredEntityNames := make(map[string]bool)

	for _, e := range graph.Entities {
		if nameMap[e.Name] {
			filteredEntities = append(filteredEntities, e)
			filteredEntityNames[e.Name] = true
		}
	}

	var filteredRelations []Relation
	for _, r := range graph.Relations {
		if filteredEntityNames[r.From] && filteredEntityNames[r.To] {
			filteredRelations = append(filteredRelations, r)
		}
	}

	return &KnowledgeGraph{
		Entities:  filteredEntities,
		Relations: filteredRelations,
	}, nil
}
