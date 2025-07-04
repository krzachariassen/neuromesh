package graph

import (
	"context"
	"testing"
	"time"

	"neuromesh/internal/logging"
)

// TestDebugNeo4jStorage - Debug test that leaves data in database
func TestDebugNeo4jStorage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Neo4j integration test in short mode")
	}

	ctx := context.Background()
	logger := logging.NewStructuredLogger(logging.LevelError)

	// Connect to Neo4j
	config := GraphConfig{
		Backend:       GraphBackendNeo4j,
		Neo4jURL:      "bolt://localhost:7687",
		Neo4jUser:     "neo4j",
		Neo4jPassword: "orchestrator123",
	}

	graph, err := NewNeo4jGraph(ctx, config, logger)
	if err != nil {
		t.Skipf("Cannot connect to Neo4j: %v", err)
	}
	defer graph.Close(ctx)

	// Clear any existing test data first
	_ = graph.ClearTestData(ctx)

	// Add a simple agent
	properties := map[string]interface{}{
		"name":       "Debug Agent",
		"status":     "online",
		"created_at": time.Now().Unix(),
	}

	err = graph.AddNode(ctx, "agent", "debug-agent-001", properties)
	if err != nil {
		t.Fatalf("Failed to add agent: %v", err)
	}

	t.Logf("✅ Agent added successfully")

	// Query back to verify
	agents, err := graph.QueryNodes(ctx, "agent", nil)
	if err != nil {
		t.Fatalf("Failed to query agents: %v", err)
	}

	t.Logf("✅ Found %d agents in database", len(agents))
	for i, agent := range agents {
		t.Logf("Agent %d: %+v", i, agent)
	}

	// DON'T CLEANUP - leave data in database for manual verification
	t.Logf("⚠️  Data left in database for manual verification")
}
