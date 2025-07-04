package registry

import (
	"context"
	"testing"
	"time"

	"neuromesh/internal/agent/domain"
	"neuromesh/internal/graph"
	"neuromesh/internal/logging"
)

// TestProductionFlowAgentPersistence tests the complete production flow
// This simulates the actual flow: orchestrator gRPC -> registry -> graph
func TestProductionFlowAgentPersistence(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Neo4j integration test in short mode")
	}

	ctx := context.Background()
	logger := logging.NewStructuredLogger(logging.LevelInfo)

	// Setup Neo4j connection (same as production)
	config := graph.GraphConfig{
		Backend:       graph.GraphBackendNeo4j,
		Neo4jURL:      "bolt://localhost:7687",
		Neo4jUser:     "neo4j",
		Neo4jPassword: "orchestrator123",
	}

	graphDB, err := graph.NewNeo4jGraph(ctx, config, logger)
	if err != nil {
		t.Skipf("Cannot connect to Neo4j: %v", err)
	}
	defer graphDB.Close(ctx)

	// Clear any existing test data
	_ = graphDB.ClearTestData(ctx)
	defer graphDB.ClearTestData(ctx)

	// Create registry service (same as production orchestrator)
	registryService := NewService(graphDB, logger)

	// === PHASE 1: Simulate agent registration (via gRPC) ===
	t.Log("🔄 PHASE 1: Simulating agent registration via gRPC...")

	testAgent := &domain.Agent{
		ID:          "text-processor-001",
		Name:        "AI-Native Text Processing Agent",
		Description: "Agent registered via gRPC",
		Status:      domain.AgentStatusOnline,
		Capabilities: []domain.AgentCapability{
			{
				Name:        "word-count",
				Description: "Count words in text",
			},
			{
				Name:        "text-analysis",
				Description: "Analyze text sentiment",
			},
		},
		Metadata:  map[string]string{"type": "text-processor"},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		LastSeen:  time.Now().UTC(),
	}

	err = registryService.RegisterAgent(ctx, testAgent)
	if err != nil {
		t.Fatalf("Failed to register agent: %v", err)
	}
	t.Log("✅ Agent registered successfully")

	// === PHASE 2: Verify agent exists in graph ===
	t.Log("🔄 PHASE 2: Verifying agent exists in graph...")

	// Check via registry service
	allAgents, err := registryService.GetAllAgents(ctx)
	if err != nil {
		t.Fatalf("Failed to get all agents: %v", err)
	}

	if len(allAgents) != 1 {
		t.Fatalf("Expected 1 agent, got %d", len(allAgents))
	}

	if allAgents[0].ID != "text-processor-001" {
		t.Fatalf("Expected agent ID 'text-processor-001', got %s", allAgents[0].ID)
	}

	t.Log("✅ Agent verified via registry service")

	// Check directly in graph
	allNodes, err := graphDB.QueryNodes(ctx, "agent", nil)
	if err != nil {
		t.Fatalf("Failed to query agents directly from graph: %v", err)
	}

	if len(allNodes) != 1 {
		t.Fatalf("Expected 1 agent node in graph, got %d", len(allNodes))
	}

	t.Log("✅ Agent verified directly in graph")

	// === PHASE 3: Simulate agent disconnect (unregister) ===
	t.Log("🔄 PHASE 3: Simulating agent disconnect...")

	err = registryService.UnregisterAgent(ctx, testAgent.ID)
	if err != nil {
		t.Fatalf("Failed to unregister agent: %v", err)
	}
	t.Log("✅ Agent unregistered successfully")

	// === PHASE 4: Verify agent persists as offline ===
	t.Log("🔄 PHASE 4: Verifying agent persists as offline...")

	// Should still exist in graph but as offline
	allAgentsAfterDisconnect, err := registryService.GetAllAgents(ctx)
	if err != nil {
		t.Fatalf("Failed to get all agents after disconnect: %v", err)
	}

	if len(allAgentsAfterDisconnect) != 1 {
		t.Fatalf("🚨 PRODUCTION BUG: Expected 1 agent after disconnect (should be offline), got %d", len(allAgentsAfterDisconnect))
	}

	if allAgentsAfterDisconnect[0].Status != domain.AgentStatusOffline {
		t.Fatalf("🚨 PRODUCTION BUG: Expected agent to be offline, got status: %s", allAgentsAfterDisconnect[0].Status)
	}

	// Verify no online agents
	onlineAgents, err := registryService.GetOnlineAgents(ctx)
	if err != nil {
		t.Fatalf("Failed to get online agents: %v", err)
	}

	if len(onlineAgents) != 0 {
		t.Fatalf("Expected 0 online agents after disconnect, got %d", len(onlineAgents))
	}

	t.Log("✅ Agent correctly persisted as offline")

	// === PHASE 5: Simulate agent reconnection ===
	t.Log("🔄 PHASE 5: Simulating agent reconnection...")

	err = registryService.RegisterAgent(ctx, testAgent)
	if err != nil {
		t.Fatalf("Failed to re-register agent: %v", err)
	}

	// Verify agent is back online
	onlineAgentsAfterReconnect, err := registryService.GetOnlineAgents(ctx)
	if err != nil {
		t.Fatalf("Failed to get online agents after reconnect: %v", err)
	}

	if len(onlineAgentsAfterReconnect) != 1 {
		t.Fatalf("Expected 1 online agent after reconnect, got %d", len(onlineAgentsAfterReconnect))
	}

	t.Log("✅ Agent successfully reconnected and online")

	// === PHASE 6: Final verification of production flow ===
	t.Log("🔄 PHASE 6: Final verification...")

	// Verify graph contains exactly one agent node
	finalNodes, err := graphDB.QueryNodes(ctx, "agent", nil)
	if err != nil {
		t.Fatalf("Failed to query final graph state: %v", err)
	}

	if len(finalNodes) != 1 {
		t.Fatalf("Expected exactly 1 agent node in final state, got %d", len(finalNodes))
	}

	t.Log("✅ PRODUCTION FLOW VERIFIED: Agent registration, disconnect, and reconnection work correctly")
	t.Log("✅ NO PRODUCTION BUG: Agents properly persist in graph throughout lifecycle")
}
