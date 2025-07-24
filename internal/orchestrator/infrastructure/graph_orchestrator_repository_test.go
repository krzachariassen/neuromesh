package infrastructure

import (
	"context"
	"testing"

	"neuromesh/internal/graph"
	"neuromesh/internal/logging"
)

func TestGraphOrchestratorRepository_EnsureSchema(t *testing.T) {
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

	repo := NewGraphOrchestratorRepository(graphInstance)

	// Test schema creation
	err = repo.EnsureSchema(ctx)
	if err != nil {
		t.Fatalf("EnsureSchema failed: %v", err)
	}

	// Verify that unique constraints were created
	t.Run("Analysis constraints", func(t *testing.T) {
		hasConstraint, err := graphInstance.HasUniqueConstraint(ctx, "Analysis", "id")
		if err != nil {
			t.Fatalf("Failed to check Analysis.id constraint: %v", err)
		}
		if !hasConstraint {
			t.Error("Analysis.id unique constraint was not created")
		}
	})

	t.Run("ExecutionPlan constraints", func(t *testing.T) {
		hasConstraint, err := graphInstance.HasUniqueConstraint(ctx, "ExecutionPlan", "id")
		if err != nil {
			t.Fatalf("Failed to check ExecutionPlan.id constraint: %v", err)
		}
		if !hasConstraint {
			t.Error("ExecutionPlan.id unique constraint was not created")
		}
	})

	t.Run("ExecutionStep constraints", func(t *testing.T) {
		hasConstraint, err := graphInstance.HasUniqueConstraint(ctx, "ExecutionStep", "id")
		if err != nil {
			t.Fatalf("Failed to check ExecutionStep.id constraint: %v", err)
		}
		if !hasConstraint {
			t.Error("ExecutionStep.id unique constraint was not created")
		}
	})

	t.Run("Decision constraints", func(t *testing.T) {
		hasConstraint, err := graphInstance.HasUniqueConstraint(ctx, "Decision", "id")
		if err != nil {
			t.Fatalf("Failed to check Decision.id constraint: %v", err)
		}
		if !hasConstraint {
			t.Error("Decision.id unique constraint was not created")
		}
	})

	// Verify that indexes were created
	t.Run("Analysis indexes", func(t *testing.T) {
		indexes := []string{"request_id", "status", "category"}
		for _, field := range indexes {
			hasIndex, err := graphInstance.HasIndex(ctx, "Analysis", field)
			if err != nil {
				t.Fatalf("Failed to check Analysis.%s index: %v", field, err)
			}
			if !hasIndex {
				t.Errorf("Analysis.%s index was not created", field)
			}
		}
	})

	t.Run("ExecutionPlan indexes", func(t *testing.T) {
		indexes := []string{"analysis_id", "status", "priority"}
		for _, field := range indexes {
			hasIndex, err := graphInstance.HasIndex(ctx, "ExecutionPlan", field)
			if err != nil {
				t.Fatalf("Failed to check ExecutionPlan.%s index: %v", field, err)
			}
			if !hasIndex {
				t.Errorf("ExecutionPlan.%s index was not created", field)
			}
		}
	})

	t.Run("ExecutionStep indexes", func(t *testing.T) {
		indexes := []string{"plan_id", "status", "assigned_agent_id", "step_number"}
		for _, field := range indexes {
			hasIndex, err := graphInstance.HasIndex(ctx, "ExecutionStep", field)
			if err != nil {
				t.Fatalf("Failed to check ExecutionStep.%s index: %v", field, err)
			}
			if !hasIndex {
				t.Errorf("ExecutionStep.%s index was not created", field)
			}
		}
	})

	t.Run("Decision indexes", func(t *testing.T) {
		indexes := []string{"request_id", "analysis_id", "plan_id", "type", "status"}
		for _, field := range indexes {
			hasIndex, err := graphInstance.HasIndex(ctx, "Decision", field)
			if err != nil {
				t.Fatalf("Failed to check Decision.%s index: %v", field, err)
			}
			if !hasIndex {
				t.Errorf("Decision.%s index was not created", field)
			}
		}
	})

	// Verify that schema nodes were created
	t.Run("Schema nodes exist", func(t *testing.T) {
		schemaNodes := map[string]string{
			"Analysis":      "schema_analysis",
			"ExecutionPlan": "schema_execution_plan",
			"ExecutionStep": "schema_execution_step",
			"Decision":      "schema_decision",
		}

		for nodeType, nodeID := range schemaNodes {
			node, err := graphInstance.GetNode(ctx, nodeType, nodeID)
			if err != nil {
				t.Fatalf("Failed to get schema %s node: %v", nodeType, err)
			}
			if node == nil {
				t.Errorf("Schema %s node was not created", nodeType)
			}
		}
	})

	// Verify that core relationships were created
	t.Run("Schema relationships exist", func(t *testing.T) {
		// Test a few key relationships
		relationships := []struct {
			fromType, fromID, toType, toID, relType string
		}{
			{"Analysis", "schema_analysis", "ExecutionPlan", "schema_execution_plan", "CREATES_PLAN"},
			{"Analysis", "schema_analysis", "Decision", "schema_decision", "INFORMS_DECISION"},
			{"ExecutionPlan", "schema_execution_plan", "ExecutionStep", "schema_execution_step", "CONTAINS_STEP"},
		}

		for _, rel := range relationships {
			hasRel, err := graphInstance.HasRelationshipType(ctx, rel.relType)
			if err != nil {
				t.Fatalf("Failed to check relationship type %s: %v", rel.relType, err)
			}
			if !hasRel {
				t.Errorf("Relationship type %s was not created", rel.relType)
			}
		}
	})
}

func TestGraphOrchestratorRepository_EnsureSchema_Idempotent(t *testing.T) {
	// Setup - create a real Neo4j connection for integration test
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

	repo := NewGraphOrchestratorRepository(graphInstance)

	// Test that running EnsureSchema multiple times doesn't cause errors
	for i := 0; i < 3; i++ {
		err := repo.EnsureSchema(ctx)
		if err != nil {
			t.Fatalf("EnsureSchema failed on iteration %d: %v", i+1, err)
		}
	}

	// Verify schema is still correctly in place
	hasConstraint, err := graphInstance.HasUniqueConstraint(ctx, "Analysis", "id")
	if err != nil {
		t.Fatalf("Failed to check Analysis.id constraint after multiple runs: %v", err)
	}
	if !hasConstraint {
		t.Error("Analysis.id unique constraint was not preserved after multiple runs")
	}
}

func TestNewGraphOrchestratorRepository(t *testing.T) {
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

	repo := NewGraphOrchestratorRepository(graphInstance)

	if repo == nil {
		t.Fatal("NewGraphOrchestratorRepository returned nil")
	}

	if repo.graph != graphInstance {
		t.Error("NewGraphOrchestratorRepository did not set graph correctly")
	}
}
