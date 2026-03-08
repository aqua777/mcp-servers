package memory

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
)

type ManagerTestSuite struct {
	suite.Suite
	tempDir  string
	tempFile string
	manager  *KnowledgeGraphManager
}

func (suite *ManagerTestSuite) SetupTest() {
	var err error
	suite.tempDir, err = os.MkdirTemp("", "memory-test")
	suite.Require().NoError(err)

	suite.tempFile = filepath.Join(suite.tempDir, "memory.jsonl")
	suite.manager, err = NewKnowledgeGraphManager(suite.tempFile)
	suite.Require().NoError(err)
}

func (suite *ManagerTestSuite) TearDownTest() {
	os.RemoveAll(suite.tempDir)
}

func (suite *ManagerTestSuite) TestNewKnowledgeGraphManager_Migration() {
	// Setup legacy memory.json
	tempDir, err := os.MkdirTemp("", "migration-test")
	suite.Require().NoError(err)
	defer os.RemoveAll(tempDir)

	oldPath := filepath.Join(tempDir, "memory.json")
	newPath := filepath.Join(tempDir, "memory.jsonl")

	err = os.WriteFile(oldPath, []byte(`{"type":"entity","name":"test","entityType":"person","observations":["obs"]}`+"\n"), 0644)
	suite.Require().NoError(err)

	m, err := NewKnowledgeGraphManager(newPath)
	suite.Require().NoError(err)
	suite.NotNil(m)

	// Check that migration happened
	_, err = os.Stat(newPath)
	suite.Require().NoError(err)

	_, err = os.Stat(oldPath)
	suite.True(os.IsNotExist(err), "old file should be removed/renamed")

	// Verify content
	graph, err := m.ReadGraph()
	suite.Require().NoError(err)
	suite.Len(graph.Entities, 1)
	suite.Equal("test", graph.Entities[0].Name)
}

func (suite *ManagerTestSuite) TestCreateEntities() {
	entities := []Entity{
		{Name: "Alice", EntityType: "Person", Observations: []string{"Likes Go"}},
		{Name: "Bob", EntityType: "Person", Observations: []string{"Likes Rust"}},
	}

	created, err := suite.manager.CreateEntities(entities)
	suite.Require().NoError(err)
	suite.Len(created, 2)
	suite.Equal("Alice", created[0].Name)

	// Test duplicate creation
	createdAgain, err := suite.manager.CreateEntities([]Entity{
		{Name: "Alice", EntityType: "Person", Observations: []string{"Different"}},
		{Name: "Charlie", EntityType: "Person", Observations: []string{"Likes C++"}},
	})
	suite.Require().NoError(err)
	suite.Len(createdAgain, 1) // Only Charlie should be created
	suite.Equal("Charlie", createdAgain[0].Name)

	graph, err := suite.manager.ReadGraph()
	suite.Require().NoError(err)
	suite.Len(graph.Entities, 3)
}

func (suite *ManagerTestSuite) TestCreateRelations() {
	suite.manager.CreateEntities([]Entity{
		{Name: "Alice", EntityType: "Person", Observations: []string{}},
		{Name: "Bob", EntityType: "Person", Observations: []string{}},
	})

	relations := []Relation{
		{From: "Alice", To: "Bob", RelationType: "knows"},
	}

	created, err := suite.manager.CreateRelations(relations)
	suite.Require().NoError(err)
	suite.Len(created, 1)

	// Test duplicate
	createdAgain, err := suite.manager.CreateRelations([]Relation{
		{From: "Alice", To: "Bob", RelationType: "knows"},
		{From: "Bob", To: "Alice", RelationType: "knows"},
	})
	suite.Require().NoError(err)
	suite.Len(createdAgain, 1) // Only Bob->Alice

	graph, err := suite.manager.ReadGraph()
	suite.Require().NoError(err)
	suite.Len(graph.Relations, 2)
}

func (suite *ManagerTestSuite) TestAddObservations() {
	suite.manager.CreateEntities([]Entity{
		{Name: "Alice", EntityType: "Person", Observations: []string{"O1"}},
	})

	reqs := []ObservationRequest{
		{EntityName: "Alice", Contents: []string{"O2", "O1", "O3"}},
	}

	res, err := suite.manager.AddObservations(reqs)
	suite.Require().NoError(err)
	suite.Len(res, 1)
	suite.ElementsMatch([]string{"O2", "O3"}, res[0].AddedObservations)

	graph, err := suite.manager.ReadGraph()
	suite.Require().NoError(err)
	suite.ElementsMatch([]string{"O1", "O2", "O3"}, graph.Entities[0].Observations)

	// Error test
	_, err = suite.manager.AddObservations([]ObservationRequest{
		{EntityName: "Nobody", Contents: []string{"O1"}},
	})
	suite.Require().Error(err)
}

func (suite *ManagerTestSuite) TestDeleteEntities() {
	suite.manager.CreateEntities([]Entity{
		{Name: "Alice", EntityType: "Person", Observations: []string{}},
		{Name: "Bob", EntityType: "Person", Observations: []string{}},
	})
	suite.manager.CreateRelations([]Relation{
		{From: "Alice", To: "Bob", RelationType: "knows"},
	})

	err := suite.manager.DeleteEntities([]string{"Alice"})
	suite.Require().NoError(err)

	graph, err := suite.manager.ReadGraph()
	suite.Require().NoError(err)
	suite.Len(graph.Entities, 1)
	suite.Equal("Bob", graph.Entities[0].Name)
	suite.Len(graph.Relations, 0) // Relation should be deleted because Alice is deleted
}

func (suite *ManagerTestSuite) TestDeleteObservations() {
	suite.manager.CreateEntities([]Entity{
		{Name: "Alice", EntityType: "Person", Observations: []string{"O1", "O2", "O3"}},
	})

	err := suite.manager.DeleteObservations([]ObservationDeletion{
		{EntityName: "Alice", Observations: []string{"O2"}},
	})
	suite.Require().NoError(err)

	graph, err := suite.manager.ReadGraph()
	suite.Require().NoError(err)
	suite.ElementsMatch([]string{"O1", "O3"}, graph.Entities[0].Observations)
}

func (suite *ManagerTestSuite) TestDeleteRelations() {
	suite.manager.CreateRelations([]Relation{
		{From: "Alice", To: "Bob", RelationType: "knows"},
		{From: "Alice", To: "Charlie", RelationType: "knows"},
	})

	err := suite.manager.DeleteRelations([]Relation{
		{From: "Alice", To: "Bob", RelationType: "knows"},
	})
	suite.Require().NoError(err)

	graph, err := suite.manager.ReadGraph()
	suite.Require().NoError(err)
	suite.Len(graph.Relations, 1)
	suite.Equal("Charlie", graph.Relations[0].To)
}

func (suite *ManagerTestSuite) TestSearchNodes() {
	suite.manager.CreateEntities([]Entity{
		{Name: "Alice", EntityType: "Person", Observations: []string{"Likes Go"}},
		{Name: "Bob", EntityType: "Person", Observations: []string{"Likes Rust"}},
	})
	suite.manager.CreateRelations([]Relation{
		{From: "Alice", To: "Bob", RelationType: "knows"},
	})

	// Search by name
	g1, err := suite.manager.SearchNodes("alice")
	suite.Require().NoError(err)
	suite.Len(g1.Entities, 1)
	suite.Equal("Alice", g1.Entities[0].Name)
	suite.Len(g1.Relations, 0) // Because Bob is not matched

	// Search by observation
	g2, err := suite.manager.SearchNodes("rust")
	suite.Require().NoError(err)
	suite.Len(g2.Entities, 1)
	suite.Equal("Bob", g2.Entities[0].Name)

	// Search by entity type (matches both)
	g3, err := suite.manager.SearchNodes("person")
	suite.Require().NoError(err)
	suite.Len(g3.Entities, 2)
	suite.Len(g3.Relations, 1) // Both matched, relation should be included
}

func (suite *ManagerTestSuite) TestOpenNodes() {
	suite.manager.CreateEntities([]Entity{
		{Name: "Alice", EntityType: "Person", Observations: []string{"Likes Go"}},
		{Name: "Bob", EntityType: "Person", Observations: []string{"Likes Rust"}},
		{Name: "Charlie", EntityType: "Person", Observations: []string{}},
	})
	suite.manager.CreateRelations([]Relation{
		{From: "Alice", To: "Bob", RelationType: "knows"},
		{From: "Alice", To: "Charlie", RelationType: "knows"},
	})

	g, err := suite.manager.OpenNodes([]string{"Alice", "Bob"})
	suite.Require().NoError(err)
	suite.Len(g.Entities, 2)
	suite.Len(g.Relations, 1)
	suite.Equal("Bob", g.Relations[0].To)
}

func (suite *ManagerTestSuite) TestCorruptJSONL() {
	err := os.WriteFile(suite.tempFile, []byte("invalid json\n{\"type\":\"entity\",\"name\":\"Alice\"}\n"), 0644)
	suite.Require().NoError(err)

	graph, err := suite.manager.ReadGraph()
	suite.Require().NoError(err)
	suite.Len(graph.Entities, 1)
	suite.Equal("Alice", graph.Entities[0].Name)
}

func (suite *ManagerTestSuite) TestInvalidSave() {
	// Using a read-only directory to force a save error
	// But in modern linux this might be tricky, let's just make the filepath a directory
	badManager, _ := NewKnowledgeGraphManager(suite.tempDir)
	
	err := badManager.saveGraph(&KnowledgeGraph{})
	suite.Require().Error(err)
}

func TestManagerSuite(t *testing.T) {
	suite.Run(t, new(ManagerTestSuite))
}
