package infrastructure

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"neuromesh/internal/planning/domain"
)

// GREEN Phase: Tests should now pass with our GraphPlanningRepository implementation
func TestGraphPlanningRepository_Store_ShouldPersistPlanningResult(t *testing.T) {
	ctx := context.Background()
	graph := setupTestGraph(t)
	repo := NewGraphPlanningRepository(graph)

	// Create test planning result
	planningResult := domain.NewExecutePlanningResult(
		"req-123",
		"medical diagnosis",
		"healthcare",
		95,
		[]string{"symptom-agent", "lab-agent"},
		[]string{"symptom-agent", "lab-agent", "diagnostic-agent"},
		"plan-456",
		"Need diagnostic agent for comprehensive analysis",
	)

	// Act: Store planning result
	err := repo.Store(ctx, planningResult)

	// Assert: Should store without error
	require.NoError(t, err)

	// Verify storage by retrieving
	retrieved, err := repo.GetByID(ctx, planningResult.ID)
	require.NoError(t, err)
	assert.Equal(t, planningResult.ID, retrieved.ID)
	assert.Equal(t, planningResult.Type, retrieved.Type)
	assert.Equal(t, planningResult.Intent, retrieved.Intent)
	assert.Equal(t, planningResult.ExecutionPlanID, retrieved.ExecutionPlanID)
	assert.Equal(t, planningResult.AgentGap, retrieved.AgentGap)
}

func TestGraphPlanningRepository_GetByRequestID_ShouldReturnPlanningHistory(t *testing.T) {
	ctx := context.Background()
	graph := setupTestGraph(t)
	repo := NewGraphPlanningRepository(graph)

	requestID := "req-123"

	// Create multiple planning results for same request (planning evolution)
	planning1 := domain.NewClarificationPlanningResult(requestID, "unclear symptoms", "healthcare", 60, "Need more symptom details", "Initial analysis unclear")

	planning2 := domain.NewExecutePlanningResult(requestID, "medical diagnosis", "healthcare", 95,
		[]string{"symptom-agent", "lab-agent", "diagnostic-agent"},
		[]string{"symptom-agent", "lab-agent", "diagnostic-agent"}, "plan-456", "All agents available")

	err := repo.Store(ctx, planning1)
	require.NoError(t, err)
	err = repo.Store(ctx, planning2)
	require.NoError(t, err)

	// Act: Get planning history
	results, err := repo.GetByRequestID(ctx, requestID)

	// Assert: Should return both planning results
	require.NoError(t, err)
	assert.Len(t, results, 2)

	// Should be ordered by timestamp (newest first)
	assert.True(t, results[0].Timestamp.After(results[1].Timestamp) || results[0].Timestamp.Equal(results[1].Timestamp))
}

func TestGraphPlanningRepository_AgentGapAnalysis_ShouldTrackAvailabilityGaps(t *testing.T) {
	ctx := context.Background()
	graph := setupTestGraph(t)
	repo := NewGraphPlanningRepository(graph)

	// Create planning result with agent gaps
	planning := domain.NewExecutePlanningResult("req-123", "complex diagnosis", "healthcare", 75,
		[]string{"symptom-agent", "lab-agent"},                                         // Available
		[]string{"symptom-agent", "lab-agent", "diagnostic-agent", "specialist-agent"}, // Required
		"", // No execution plan due to missing agents
		"Missing diagnostic and specialist agents")

	err := repo.Store(ctx, planning)
	require.NoError(t, err)

	// Retrieve and verify agent gap analysis
	retrieved, err := repo.GetByID(ctx, planning.ID)
	require.NoError(t, err)

	expectedGap := []string{"diagnostic-agent", "specialist-agent"}
	assert.Equal(t, expectedGap, retrieved.AgentGap)
	assert.Empty(t, retrieved.ExecutionPlanID, "Should not have execution plan when agents missing")
}

func TestGraphPlanningRepository_EnsureSchema_ShouldCreateConstraintsAndIndexes(t *testing.T) {
	ctx := context.Background()
	graph := setupTestGraph(t)
	repo := NewGraphPlanningRepository(graph)

	// Should not error when ensuring schema
	err := repo.EnsureSchema(ctx)
	assert.NoError(t, err)

	// Should be idempotent
	err = repo.EnsureSchema(ctx)
	assert.NoError(t, err)
}
