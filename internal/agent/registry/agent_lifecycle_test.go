package registry

import (
	"context"
	"testing"
	"time"

	"neuromesh/internal/agent/domain"
	"neuromesh/internal/graph"
	"neuromesh/internal/logging"
)

// TestAgentLifecyclePersistence tests the complete agent lifecycle
// This test will prove that agents are persisted properly through registration and unregistration
func TestAgentLifecyclePersistence(t *testing.T) {
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

	// Clean up any existing test data
	_ = graphInstance.ClearTestData(ctx)

	// Create agent registry service
	registryService := NewService(graphInstance, logger)

	// Create a test agent
	testAgent := &domain.Agent{
		ID:          "test-agent-lifecycle",
		Name:        "Test Agent Lifecycle",
		Description: "Test agent for lifecycle testing",
		Status:      domain.AgentStatusOnline,
		Capabilities: []domain.AgentCapability{
			{
				Name:        "text-processing",
				Description: "Process text data",
			},
		},
		Metadata:  map[string]string{"test": "lifecycle"},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		LastSeen:  time.Now().UTC(),
	}

	// Phase 1: Register agent
	t.Log("🔄 Phase 1: Registering agent...")
	err = registryService.RegisterAgent(ctx, testAgent)
	if err != nil {
		t.Fatalf("Failed to register agent: %v", err)
	}
	t.Log("✅ Agent registered successfully")

	// Verify agent is in graph and online
	allAgents, err := registryService.GetAllAgents(ctx)
	if err != nil {
		t.Fatalf("Failed to get all agents: %v", err)
	}

	if len(allAgents) != 1 {
		t.Fatalf("Expected 1 agent after registration, got %d", len(allAgents))
	}

	if allAgents[0].Status != domain.AgentStatusOnline {
		t.Fatalf("Expected agent to be online, got status: %s", allAgents[0].Status)
	}

	t.Log("✅ Agent verified in graph as online")

	// Phase 2: Unregister agent (simulate disconnect)
	t.Log("🔄 Phase 2: Unregistering agent (simulating disconnect)...")
	err = registryService.UnregisterAgent(ctx, testAgent.ID)
	if err != nil {
		t.Fatalf("Failed to unregister agent: %v", err)
	}
	t.Log("✅ Agent unregistered successfully")

	// Phase 3: Verify agent is still in graph but marked as offline
	t.Log("🔄 Phase 3: Verifying agent persistence...")
	allAgentsAfterUnregister, err := registryService.GetAllAgents(ctx)
	if err != nil {
		t.Fatalf("Failed to get all agents after unregister: %v", err)
	}

	if len(allAgentsAfterUnregister) != 1 {
		t.Fatalf("🚨 CRITICAL: Expected 1 agent after unregistration (should be offline), got %d", len(allAgentsAfterUnregister))
	}

	agentAfterUnregister := allAgentsAfterUnregister[0]
	if agentAfterUnregister.Status != domain.AgentStatusOffline {
		t.Fatalf("🚨 CRITICAL: Expected agent to be offline after unregistration, got status: %s", agentAfterUnregister.Status)
	}

	t.Log("✅ Agent persistence verified: Agent exists in graph as offline")

	// Phase 4: Verify online agents query excludes offline agents
	t.Log("🔄 Phase 4: Verifying online agents query...")
	onlineAgents, err := registryService.GetOnlineAgents(ctx)
	if err != nil {
		t.Fatalf("Failed to get online agents: %v", err)
	}

	if len(onlineAgents) != 0 {
		t.Fatalf("Expected 0 online agents, got %d", len(onlineAgents))
	}

	t.Log("✅ Online agents query correctly excludes offline agents")

	// Phase 5: Re-register agent (simulate reconnection)
	t.Log("🔄 Phase 5: Re-registering agent (simulating reconnection)...")
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

	if onlineAgentsAfterReconnect[0].Status != domain.AgentStatusOnline {
		t.Fatalf("Expected agent to be online after reconnect, got status: %s", onlineAgentsAfterReconnect[0].Status)
	}

	t.Log("✅ Agent lifecycle test complete: Registration, unregistration, and re-registration all work correctly")

	// Clean up
	_ = graphInstance.ClearTestData(ctx)
}
