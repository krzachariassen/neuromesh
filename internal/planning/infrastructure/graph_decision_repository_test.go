package infrastructure

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"neuromesh/internal/graph"
	"neuromesh/internal/planning/domain"
)

func TestGraphDecisionRepository_Store_Success(t *testing.T) {
	graph, cleanup := setupTestNeo4j(t)
	defer cleanup()

	repo := NewGraphDecisionRepository(graph)
	ctx := context.Background()

	// Create test decision
	decision := domain.NewClarifyDecision(
		"req-123",
		"analysis-456",
		"Could you clarify what you mean?",
		"Request is ambiguous",
	)

	// Store decision should succeed
	err := repo.Store(ctx, decision)
	require.NoError(t, err)

	// Verify decision was stored
	retrieved, err := repo.GetByID(ctx, decision.ID)
	require.NoError(t, err)
	assert.Equal(t, decision.ID, retrieved.ID)
	assert.Equal(t, decision.RequestID, retrieved.RequestID)
	assert.Equal(t, decision.Type, retrieved.Type)
	assert.Equal(t, decision.ClarificationQuestion, retrieved.ClarificationQuestion)
}

func TestGraphDecisionRepository_Store_ExecuteDecision(t *testing.T) {
	graph, cleanup := setupTestNeo4j(t)
	defer cleanup()

	repo := NewGraphDecisionRepository(graph)
	ctx := context.Background()

	// Create test execute decision
	decision := domain.NewExecuteDecision(
		"req-789",
		"analysis-101",
		"plan-202",
		"Use text-processor agent",
		"Request is clear, can execute",
	)

	// Store decision should succeed
	err := repo.Store(ctx, decision)
	require.NoError(t, err)

	// Verify decision was stored
	retrieved, err := repo.GetByID(ctx, decision.ID)
	require.NoError(t, err)
	assert.Equal(t, decision.ID, retrieved.ID)
	assert.Equal(t, decision.ExecutionPlanID, retrieved.ExecutionPlanID)
	assert.Equal(t, domain.DecisionTypeExecute, retrieved.Type)
}

func TestGraphDecisionRepository_GetByRequestID_Success(t *testing.T) {
	graph, cleanup := setupTestNeo4j(t)
	defer cleanup()

	repo := NewGraphDecisionRepository(graph)
	ctx := context.Background()

	// Store a decision
	decision := domain.NewClarifyDecision(
		"unique-request-id",
		"analysis-123",
		"Please clarify",
		"Need more info",
	)
	err := repo.Store(ctx, decision)
	require.NoError(t, err)

	// Retrieve by request ID
	retrieved, err := repo.GetByRequestID(ctx, "unique-request-id")
	require.NoError(t, err)
	assert.Equal(t, decision.ID, retrieved.ID)
	assert.Equal(t, "unique-request-id", retrieved.RequestID)
}

func TestGraphDecisionRepository_GetByType_Success(t *testing.T) {
	graph, cleanup := setupTestNeo4j(t)
	defer cleanup()

	repo := NewGraphDecisionRepository(graph)
	ctx := context.Background()

	// Store multiple decisions of different types
	clarifyDecision := domain.NewClarifyDecision("req-1", "ana-1", "Clarify please", "Unclear request")
	executeDecision := domain.NewExecuteDecision("req-2", "ana-2", "plan-1", "Execute now", "Clear path")

	err := repo.Store(ctx, clarifyDecision)
	require.NoError(t, err)
	err = repo.Store(ctx, executeDecision)
	require.NoError(t, err)

	// Get clarify decisions
	clarifyDecisions, err := repo.GetByType(ctx, domain.DecisionTypeClarify, 10)
	require.NoError(t, err)
	assert.Len(t, clarifyDecisions, 1)
	assert.Equal(t, domain.DecisionTypeClarify, clarifyDecisions[0].Type)

	// Get execute decisions
	executeDecisions, err := repo.GetByType(ctx, domain.DecisionTypeExecute, 10)
	require.NoError(t, err)
	assert.Len(t, executeDecisions, 1)
	assert.Equal(t, domain.DecisionTypeExecute, executeDecisions[0].Type)
}

func TestGraphDecisionRepository_LinkToAnalysis_Success(t *testing.T) {
	graph, cleanup := setupTestNeo4j(t)
	defer cleanup()

	repo := NewGraphDecisionRepository(graph)
	ctx := context.Background()

	// Create test analysis first
	createTestAnalysis(t, graph, "analysis-1", "req-1")

	// Store decision
	decision := domain.NewClarifyDecision("req-1", "analysis-1", "Need clarification", "Unclear")
	err := repo.Store(ctx, decision)
	require.NoError(t, err)

	// Link to analysis
	err = repo.LinkToAnalysis(ctx, decision.ID, "analysis-1")
	require.NoError(t, err)

	// Verify relationship exists by querying
	retrieved, err := repo.GetByAnalysisID(ctx, "analysis-1")
	require.NoError(t, err)
	assert.Equal(t, decision.ID, retrieved.ID)
}

func TestGraphDecisionRepository_LinkToExecutionPlan_Success(t *testing.T) {
	graph, cleanup := setupTestNeo4j(t)
	defer cleanup()

	repo := NewGraphDecisionRepository(graph)
	ctx := context.Background()

	// Create test execution plan first
	createTestExecutionPlan(t, graph, "plan-123")

	// Store execute decision
	decision := domain.NewExecuteDecision("req-1", "analysis-1", "plan-123", "coordination", "reasoning")
	err := repo.Store(ctx, decision)
	require.NoError(t, err)

	// Link to execution plan
	err = repo.LinkToExecutionPlan(ctx, decision.ID, "plan-123")
	require.NoError(t, err)

	// Verify decision references the plan
	retrieved, err := repo.GetByID(ctx, decision.ID)
	require.NoError(t, err)
	assert.Equal(t, "plan-123", retrieved.ExecutionPlanID)
}

// Helper functions

func createTestAnalysis(t *testing.T, g graph.Graph, analysisID, requestID string) {
	ctx := context.Background()

	properties := map[string]interface{}{
		"id":              analysisID,
		"request_id":      requestID,
		"intent":          "test intent",
		"category":        "test",
		"confidence":      80,
		"required_agents": "[]",
		"reasoning":       "test reasoning",
		"timestamp":       time.Now().UTC(),
		"created_at":      time.Now().UTC(),
	}

	err := g.AddNode(ctx, "Analysis", analysisID, properties)
	require.NoError(t, err, "Failed to create test analysis")
}

func createTestExecutionPlan(t *testing.T, g graph.Graph, planID string) {
	ctx := context.Background()

	properties := map[string]interface{}{
		"id":          planID,
		"name":        "Test Plan",
		"description": "Test execution plan",
		"priority":    "MEDIUM",
		"status":      "PENDING",
		"created_at":  time.Now().UTC(),
	}

	err := g.AddNode(ctx, "ExecutionPlan", planID, properties)
	require.NoError(t, err, "Failed to create test execution plan")
}
