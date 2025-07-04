package graph

import (
	"context"
	"testing"
	"time"

	"neuromesh/internal/logging"
)

// TestNeo4jAgentStorageAndRetrieval tests the complete agent lifecycle in Neo4j
// This test should FAIL initially to prove our graph storage is broken
func TestNeo4jAgentStorageAndRetrieval(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Neo4j integration test in short mode")
	}

	ctx := context.Background()
	logger := logging.NewStructuredLogger(logging.LevelError)

	// Try to connect to real Neo4j
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

	// Clear any existing test data
	_ = graph.ClearTestData(ctx)
	defer graph.ClearTestData(ctx) // Cleanup after test

	// Test 1: Store multiple agents with different statuses
	agents := []struct {
		id           string
		name         string
		status       string
		capabilities []string
	}{
		{"text-processor-001", "Text Processor", "online", []string{"word-count", "text-analysis"}},
		{"image-processor-001", "Image Processor", "online", []string{"image-resize", "format-convert"}},
		{"data-processor-001", "Data Processor", "offline", []string{"data-transform", "validation"}},
	}

	// Store agents in graph
	for _, agent := range agents {
		properties := map[string]interface{}{
			"name":         agent.name,
			"status":       agent.status,
			"capabilities": agent.capabilities,
			"created_at":   time.Now().Unix(),
		}

		err := graph.AddNode(ctx, "agent", agent.id, properties)
		if err != nil {
			t.Fatalf("Failed to add agent %s: %v", agent.id, err)
		}
	}

	// Test 2: Query ALL agents (should return 3)
	allAgents, err := graph.QueryNodes(ctx, "agent", nil)
	if err != nil {
		t.Fatalf("Failed to query all agents: %v", err)
	}

	if len(allAgents) != 3 {
		t.Errorf("Expected 3 agents, got %d", len(allAgents))
		t.Logf("Agents found: %+v", allAgents)
	}

	// Test 3: Query ONLINE agents only (should return 2)
	onlineAgents, err := graph.QueryNodes(ctx, "agent", map[string]interface{}{
		"status": "online",
	})
	if err != nil {
		t.Fatalf("Failed to query online agents: %v", err)
	}

	if len(onlineAgents) != 2 {
		t.Errorf("Expected 2 online agents, got %d", len(onlineAgents))
		t.Logf("Online agents found: %+v", onlineAgents)
	}

	// Test 4: Query with multiple filters (should return 1)
	specificAgent, err := graph.QueryNodes(ctx, "agent", map[string]interface{}{
		"status": "online",
		"name":   "Text Processor",
	})
	if err != nil {
		t.Fatalf("Failed to query specific agent: %v", err)
	}

	if len(specificAgent) != 1 {
		t.Errorf("Expected 1 specific agent, got %d", len(specificAgent))
		t.Logf("Specific agents found: %+v", specificAgent)
	}

	// Test 5: Verify agent properties are preserved
	if len(specificAgent) > 0 {
		agent := specificAgent[0]
		if agent["id"] != "text-processor-001" {
			t.Errorf("Expected agent ID 'text-processor-001', got %v", agent["id"])
		}
		if agent["name"] != "Text Processor" {
			t.Errorf("Expected agent name 'Text Processor', got %v", agent["name"])
		}
		if agent["status"] != "online" {
			t.Errorf("Expected agent status 'online', got %v", agent["status"])
		}
	}
}

// TestGraphAgentServiceIntegration tests the GraphAgentService with real graph
// This simulates the exact flow that's broken in production
func TestGraphAgentServiceIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Neo4j integration test in short mode")
	}

	ctx := context.Background()
	logger := logging.NewStructuredLogger(logging.LevelError)

	// Create graph
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

	// Clear test data
	_ = graph.ClearTestData(ctx)
	defer graph.ClearTestData(ctx)

	// Simulate the EXACT agent registration process from registry service
	// Neo4j requires primitives or JSON strings for complex data
	capabilitiesJSON := `[{"name":"word-count","description":"Count words"},{"name":"text-analysis","description":"Analyze text"}]`

	agentProperties := map[string]interface{}{
		"name":         "AI-Native Text Processing Agent",
		"description":  "Agent registered via gRPC",
		"status":       "online",
		"created_at":   time.Now().Unix(),
		"updated_at":   time.Now().Unix(),
		"last_seen":    time.Now().Unix(),
		"capabilities": capabilitiesJSON,
	}

	// Store agent (simulating registry service)
	err = graph.AddNode(ctx, "agent", "text-processor-001", agentProperties)
	if err != nil {
		t.Fatalf("Failed to store agent: %v", err)
	}

	// Query agent (simulating GraphAgentService.GetAvailableAgents)
	// This is the EXACT query that's failing in production
	agents, err := graph.QueryNodes(ctx, "agent", map[string]interface{}{
		"status": "online",
	})
	if err != nil {
		t.Fatalf("Failed to query agents: %v", err)
	}

	// This should find the agent but currently FAILS
	if len(agents) != 1 {
		t.Errorf("PRODUCTION BUG: Expected 1 agent, got %d", len(agents))
		t.Logf("Query result: %+v", agents)

		// Debug: Try querying without filters
		allAgents, _ := graph.QueryNodes(ctx, "agent", nil)
		t.Logf("All agents in graph: %+v", allAgents)

		// This test proves the bug exists
		t.Fatal("This test exposes the production bug where agents are not found despite being stored")
	}
}
