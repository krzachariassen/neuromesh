package registry

import (
	"context"
	"testing"
	"time"

	"neuromesh/internal/agent/domain"
	"neuromesh/internal/graph"
	"neuromesh/internal/logging"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// TestRealNeo4jPersistence verifies that agents are actually persisted in the real Neo4j database
// This test executes direct Cypher queries to validate data persistence at the database level
func TestRealNeo4jPersistence(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Neo4j integration test in short mode")
	}

	ctx := context.Background()
	logger := logging.NewStructuredLogger(logging.LevelInfo)

	// Connect to Neo4j - same as production
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
	// NOTE: Defer cleanup commented out for manual inspection of data
	// defer graphInstance.ClearTestData(ctx)

	// Create agent registry service
	registryService := NewService(graphInstance, logger)

	// === PHASE 1: Register agent and verify in Neo4j directly ===
	t.Log("🔄 PHASE 1: Registering agent and verifying in real Neo4j...")

	testAgent := &domain.Agent{
		ID:          "real-neo4j-test-agent",
		Name:        "Real Neo4j Test Agent",
		Description: "Agent for testing real Neo4j persistence",
		Status:      domain.AgentStatusOnline,
		Capabilities: []domain.AgentCapability{
			{
				Name:        "neo4j-testing",
				Description: "Test Neo4j persistence",
			},
		},
		Metadata:  map[string]string{"test_type": "real_neo4j"},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		LastSeen:  time.Now().UTC(),
	}

	// Register agent through our application layer
	err = registryService.RegisterAgent(ctx, testAgent)
	if err != nil {
		t.Fatalf("Failed to register agent: %v", err)
	}
	t.Log("✅ Agent registered through application layer")

	// Now verify directly in Neo4j using Cypher query
	session := graphInstance.Driver().NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	// Execute direct Cypher query to find the agent
	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		cypherQuery := `
			MATCH (a:agent {id: $agentId})
			RETURN a.id as id, 
			       a.name as name, 
			       a.description as description,
			       a.status as status,
			       a.capabilities as capabilities,
			       a.metadata as metadata,
			       a.created_at as created_at,
			       a.updated_at as updated_at
		`
		result, err := tx.Run(ctx, cypherQuery, map[string]interface{}{
			"agentId": testAgent.ID,
		})
		if err != nil {
			return nil, err
		}

		if result.Next(ctx) {
			record := result.Record()
			return map[string]interface{}{
				"id":           record.Values[0],
				"name":         record.Values[1],
				"description":  record.Values[2],
				"status":       record.Values[3],
				"capabilities": record.Values[4],
				"metadata":     record.Values[5],
				"created_at":   record.Values[6],
				"updated_at":   record.Values[7],
			}, nil
		}

		return nil, nil // Agent not found
	})

	if err != nil {
		t.Fatalf("Failed to execute direct Neo4j query: %v", err)
	}

	if result == nil {
		t.Fatal("🚨 CRITICAL: Agent not found in Neo4j database via direct Cypher query!")
	}

	agentData := result.(map[string]interface{})
	t.Logf("✅ Agent found in Neo4j via direct query: %+v", agentData)

	// Validate agent properties in Neo4j
	if agentData["id"] != testAgent.ID {
		t.Fatalf("Expected agent ID %s, got %v", testAgent.ID, agentData["id"])
	}

	if agentData["name"] != testAgent.Name {
		t.Fatalf("Expected agent name %s, got %v", testAgent.Name, agentData["name"])
	}

	if agentData["status"] != string(domain.AgentStatusOnline) {
		t.Fatalf("Expected agent status %s, got %v", domain.AgentStatusOnline, agentData["status"])
	}

	t.Log("✅ Agent properties verified in Neo4j database")

	// === PHASE 2: Unregister agent and verify it's marked offline (not deleted) ===
	t.Log("🔄 PHASE 2: Unregistering agent and verifying offline status in Neo4j...")

	err = registryService.UnregisterAgent(ctx, testAgent.ID)
	if err != nil {
		t.Fatalf("Failed to unregister agent: %v", err)
	}
	t.Log("✅ Agent unregistered through application layer")

	// Verify agent still exists in Neo4j but is marked offline
	resultAfterUnregister, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		cypherQuery := `
			MATCH (a:agent {id: $agentId})
			RETURN a.status as status, a.id as id
		`
		result, err := tx.Run(ctx, cypherQuery, map[string]interface{}{
			"agentId": testAgent.ID,
		})
		if err != nil {
			return nil, err
		}

		if result.Next(ctx) {
			record := result.Record()
			return map[string]interface{}{
				"status": record.Values[0],
				"id":     record.Values[1],
			}, nil
		}

		return nil, nil // Agent not found
	})

	if err != nil {
		t.Fatalf("Failed to execute direct Neo4j query after unregister: %v", err)
	}

	if resultAfterUnregister == nil {
		t.Fatal("🚨 CRITICAL: Agent was deleted from Neo4j instead of being marked offline!")
	}

	offlineAgentData := resultAfterUnregister.(map[string]interface{})
	if offlineAgentData["status"] != string(domain.AgentStatusOffline) {
		t.Fatalf("🚨 CRITICAL: Expected agent status offline, got %v", offlineAgentData["status"])
	}

	t.Log("✅ Agent correctly marked as offline in Neo4j (not deleted)")

	// === PHASE 3: Count total agents in Neo4j ===
	t.Log("🔄 PHASE 3: Verifying agent count in Neo4j...")

	countResult, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		cypherQuery := `MATCH (a:agent) RETURN count(a) as total`
		result, err := tx.Run(ctx, cypherQuery, nil)
		if err != nil {
			return nil, err
		}

		if result.Next(ctx) {
			record := result.Record()
			return record.Values[0], nil
		}

		return 0, nil
	})

	if err != nil {
		t.Fatalf("Failed to count agents in Neo4j: %v", err)
	}

	totalAgents := countResult.(int64)
	if totalAgents != 1 {
		t.Fatalf("Expected exactly 1 agent in Neo4j, got %d", totalAgents)
	}

	t.Log("✅ Correct agent count in Neo4j database")

	// === PHASE 4: Re-register agent and verify update (not duplicate) ===
	t.Log("🔄 PHASE 4: Re-registering agent and verifying no duplicates...")

	err = registryService.RegisterAgent(ctx, testAgent)
	if err != nil {
		t.Fatalf("Failed to re-register agent: %v", err)
	}

	// Verify we still have only 1 agent in Neo4j (no duplicates)
	countAfterReregister, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		cypherQuery := `MATCH (a:agent) RETURN count(a) as total`
		result, err := tx.Run(ctx, cypherQuery, nil)
		if err != nil {
			return nil, err
		}

		if result.Next(ctx) {
			record := result.Record()
			return record.Values[0], nil
		}

		return 0, nil
	})

	if err != nil {
		t.Fatalf("Failed to count agents after re-register: %v", err)
	}

	totalAfterReregister := countAfterReregister.(int64)
	if totalAfterReregister != 1 {
		t.Fatalf("🚨 CRITICAL: Expected 1 agent after re-register, got %d (duplicates created!)", totalAfterReregister)
	}

	// Verify agent is back online
	statusAfterReregister, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		cypherQuery := `MATCH (a:agent {id: $agentId}) RETURN a.status as status`
		result, err := tx.Run(ctx, cypherQuery, map[string]interface{}{
			"agentId": testAgent.ID,
		})
		if err != nil {
			return nil, err
		}

		if result.Next(ctx) {
			record := result.Record()
			return record.Values[0], nil
		}

		return nil, nil
	})

	if err != nil {
		t.Fatalf("Failed to check agent status after re-register: %v", err)
	}

	if statusAfterReregister != string(domain.AgentStatusOnline) {
		t.Fatalf("Expected agent to be online after re-register, got %v", statusAfterReregister)
	}

	t.Log("✅ Agent correctly updated to online status (no duplicates)")

	t.Log("🎉 REAL NEO4J PERSISTENCE VERIFIED:")
	t.Log("   ✅ Agents are persisted in real Neo4j database")
	t.Log("   ✅ Agent properties are correctly stored")
	t.Log("   ✅ Unregistration marks agents offline (doesn't delete)")
	t.Log("   ✅ Re-registration updates existing agent (no duplicates)")
	t.Log("   ✅ Production bug is FIXED!")
}
