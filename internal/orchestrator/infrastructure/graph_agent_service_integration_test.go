package infrastructure

import (
	"context"
	"testing"
	"time"

	"neuromesh/internal/graph"
	"neuromesh/internal/logging"
)

// TestProductionBugReproduction - Reproduces the exact production bug
// This test proves that agents are stored but GraphAgentService can't find them
func TestProductionBugReproduction(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Neo4j integration test in short mode")
	}

	ctx := context.Background()
	logger := logging.NewStructuredLogger(logging.LevelError)

	// Connect to Neo4j
	config := graph.GraphConfig{
		Backend:       graph.GraphBackendNeo4j,
		Neo4jURL:      "bolt://localhost:7687",
		Neo4jUser:     "neo4j",
		Neo4jPassword: "orchestrator123",
	}

	graphInstance, err := graph.NewNeo4jGraph(ctx, config, logger)
	if err != nil {
		t.Skipf("Cannot connect to Neo4j: %v", err)
	}
	defer graphInstance.Close(ctx)

	// Clear any existing test data
	_ = graphInstance.ClearTestData(ctx)
	defer graphInstance.ClearTestData(ctx)

	// STEP 1: Simulate agent registration (exactly like registry service does)
	t.Log("🔧 STEP 1: Simulating agent registration...")
	agentProperties := map[string]interface{}{
		"name":         "AI-Native Text Processing Agent",
		"description":  "Agent registered via gRPC",
		"status":       "online",
		"capabilities": `[{"name":"word-count","description":"Count words"},{"name":"text-analysis","description":"Analyze text"}]`,
		"last_seen":    time.Now().UTC(),
		"metadata":     "{}",
		"created_at":   time.Now().UTC(),
		"updated_at":   time.Now().UTC(),
	}

	err = graphInstance.AddNode(ctx, "agent", "text-processor-001", agentProperties)
	if err != nil {
		t.Fatalf("Failed to register agent: %v", err)
	}
	t.Log("✅ Agent registered successfully")

	// STEP 2: Verify agent is in database using direct query
	t.Log("🔧 STEP 2: Verifying agent exists in database...")
	allAgents, err := graphInstance.QueryNodes(ctx, "agent", nil)
	if err != nil {
		t.Fatalf("Failed to query all agents: %v", err)
	}

	if len(allAgents) != 1 {
		t.Fatalf("Expected 1 agent in database, got %d", len(allAgents))
	}
	t.Logf("✅ Agent found in database: %+v", allAgents[0])

	// STEP 3: Test the exact query that GraphAgentService uses
	t.Log("🔧 STEP 3: Testing GraphAgentService query...")
	onlineAgents, err := graphInstance.QueryNodes(ctx, "agent", map[string]interface{}{
		"status": "online",
	})
	if err != nil {
		t.Fatalf("Failed to query online agents: %v", err)
	}

	if len(onlineAgents) != 1 {
		t.Errorf("🚨 PRODUCTION BUG: Expected 1 online agent, got %d", len(onlineAgents))
		t.Logf("Query result: %+v", onlineAgents)
		t.Fatal("This exposes the production bug!")
	}
	t.Log("✅ GraphAgentService query works correctly")

	// STEP 4: Test the actual GraphAgentService
	t.Log("🔧 STEP 4: Testing actual GraphAgentService...")
	agentService := NewGraphAgentService(graphInstance)

	availableAgents, err := agentService.GetAvailableAgents(ctx)
	if err != nil {
		t.Fatalf("GraphAgentService.GetAvailableAgents failed: %v", err)
	}

	if len(availableAgents) != 1 {
		t.Errorf("🚨 PRODUCTION BUG: GraphAgentService found %d agents, expected 1", len(availableAgents))
		t.Logf("Available agents: %+v", availableAgents)
		t.Fatal("This proves the production bug exists!")
	}

	t.Log("✅ GraphAgentService works correctly")
	t.Logf("Found agent: %+v", availableAgents[0])
}
