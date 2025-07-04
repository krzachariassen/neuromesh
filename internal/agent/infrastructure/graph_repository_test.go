package infrastructure

import (
	"context"
	"testing"

	"neuromesh/internal/graph"
	"neuromesh/internal/logging"
)

// TestGraphAgentRepository_EnsureSchema tests that the repository can define and write schema
// (constraints, indexes) for Agent and Capability nodes and relationships in Neo4j
func TestGraphAgentRepository_EnsureSchema(t *testing.T) {
	// Setup - create a real Neo4j connection for integration test
	// Following the instruction: "never mock the AI provide in any tests, always use the real AI provider"
	// We extend this principle to database connections - use real Neo4j for accurate testing
	ctx := context.Background()
	logger := logging.NewStructuredLogger(logging.LevelInfo)

	config := graph.GraphConfig{
		Backend:       graph.GraphBackendNeo4j,
		Neo4jURL:      "bolt://localhost:7687",
		Neo4jUser:     "neo4j",
		Neo4jPassword: "orchestrator123",
	}

	graphInstance, err := graph.NewNeo4jGraph(ctx, config, logger)
	if err != nil {
		t.Skipf("Neo4j not available for integration test: %v", err)
	}
	defer graphInstance.Close(ctx)

	// Create repository
	repo := NewGraphAgentRepository(graphInstance)

	// RED: This should fail because EnsureSchema method doesn't exist yet
	err = repo.EnsureSchema(ctx)
	if err != nil {
		t.Fatalf("EnsureSchema failed: %v", err)
	}

	// After schema is ensured, verify that constraints and indexes are created
	// We'll verify this by checking if the schema elements exist in Neo4j

	// Test 1: Agent node should have unique constraint on ID
	hasAgentIdConstraint, err := repo.hasUniqueConstraint(ctx, "agent", "id")
	if err != nil {
		t.Fatalf("Failed to check Agent ID constraint: %v", err)
	}
	if !hasAgentIdConstraint {
		t.Error("Expected Agent node to have unique constraint on ID")
	}

	// Test 2: Agent node should have index on status for efficient status-based queries
	hasAgentStatusIndex, err := repo.hasIndex(ctx, "agent", "status")
	if err != nil {
		t.Fatalf("Failed to check Agent status index: %v", err)
	}
	if !hasAgentStatusIndex {
		t.Error("Expected Agent node to have index on status")
	}

	// Test 3: Capability node should have unique constraint on name within agent scope
	hasCapabilityConstraint, err := repo.hasUniqueConstraint(ctx, "capability", "name")
	if err != nil {
		t.Fatalf("Failed to check Capability name constraint: %v", err)
	}
	if !hasCapabilityConstraint {
		t.Error("Expected Capability node to have unique constraint on name")
	}

	// Test 4: HAS_CAPABILITY relationship should exist as defined relationship type
	hasRelationshipType, err := repo.hasRelationshipType(ctx, "HAS_CAPABILITY")
	if err != nil {
		t.Fatalf("Failed to check HAS_CAPABILITY relationship type: %v", err)
	}
	if !hasRelationshipType {
		t.Error("Expected HAS_CAPABILITY relationship type to be defined")
	}
}

// TestGraphAgentRepository_EnsureSchema_Idempotent tests that calling EnsureSchema multiple times
// is safe and doesn't cause errors
func TestGraphAgentRepository_EnsureSchema_Idempotent(t *testing.T) {
	ctx := context.Background()
	logger := logging.NewStructuredLogger(logging.LevelInfo)

	config := graph.GraphConfig{
		Backend:       graph.GraphBackendNeo4j,
		Neo4jURL:      "bolt://localhost:7687",
		Neo4jUser:     "neo4j",
		Neo4jPassword: "orchestrator123",
	}

	graphInstance, err := graph.NewNeo4jGraph(ctx, config, logger)
	if err != nil {
		t.Skipf("Neo4j not available for integration test: %v", err)
	}
	defer graphInstance.Close(ctx)

	repo := NewGraphAgentRepository(graphInstance)

	// Call EnsureSchema multiple times - should not fail
	for i := 0; i < 3; i++ {
		err = repo.EnsureSchema(ctx)
		if err != nil {
			t.Fatalf("EnsureSchema failed on iteration %d: %v", i+1, err)
		}
	}
}
