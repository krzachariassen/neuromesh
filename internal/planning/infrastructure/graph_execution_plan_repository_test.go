package infrastructure

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"neuromesh/internal/planning/domain"
)

func TestGraphExecutionPlanRepository_Create(t *testing.T) {
	ctx := context.Background()
	graph := setupTestGraph(t)
	repo := NewGraphExecutionPlanRepository(graph)

	plan := domain.NewExecutionPlan("Test Plan", "Test description", domain.ExecutionPlanPriorityHigh)
	step1 := domain.NewExecutionStep("Step 1", "First step", "agent-1")
	step2 := domain.NewExecutionStep("Step 2", "Second step", "agent-2")

	plan.AddStep(step1)
	plan.AddStep(step2)

	err := repo.Create(ctx, plan)
	require.NoError(t, err)

	// Verify plan was created
	retrievedPlan, err := repo.GetByID(ctx, plan.ID)
	require.NoError(t, err)
	assert.Equal(t, plan.ID, retrievedPlan.ID)
	assert.Equal(t, plan.Name, retrievedPlan.Name)
	assert.Equal(t, plan.Status, retrievedPlan.Status)
	assert.Equal(t, plan.Priority, retrievedPlan.Priority)

	// Verify steps were created with relationships
	steps, err := repo.GetStepsByPlanID(ctx, plan.ID)
	require.NoError(t, err)
	assert.Len(t, steps, 2)

	// Check step ordering
	assert.Equal(t, 1, steps[0].StepNumber)
	assert.Equal(t, 2, steps[1].StepNumber)
}

func TestGraphExecutionPlanRepository_Create_ValidationError(t *testing.T) {
	ctx := context.Background()
	graph := setupTestGraph(t)
	repo := NewGraphExecutionPlanRepository(graph)

	// Create invalid plan
	plan := &domain.ExecutionPlan{
		Name: "", // Invalid - empty name
	}

	err := repo.Create(ctx, plan)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid execution plan")
}

func TestGraphExecutionPlanRepository_GetByID(t *testing.T) {
	ctx := context.Background()
	graph := setupTestGraph(t)
	repo := NewGraphExecutionPlanRepository(graph)

	// Create test plan
	plan := domain.NewExecutionPlan("Test Plan", "Description", domain.ExecutionPlanPriorityMedium)
	err := repo.Create(ctx, plan)
	require.NoError(t, err)

	// Retrieve by ID
	retrievedPlan, err := repo.GetByID(ctx, plan.ID)
	require.NoError(t, err)
	assert.Equal(t, plan.ID, retrievedPlan.ID)
	assert.Equal(t, plan.Name, retrievedPlan.Name)
	assert.Equal(t, plan.Description, retrievedPlan.Description)
}

func TestGraphExecutionPlanRepository_GetByID_NotFound(t *testing.T) {
	ctx := context.Background()
	graph := setupTestGraph(t)
	repo := NewGraphExecutionPlanRepository(graph)

	plan, err := repo.GetByID(ctx, "non-existent-id")
	assert.Error(t, err)
	assert.Nil(t, plan)
	assert.Contains(t, err.Error(), "not found")
}

func TestGraphExecutionPlanRepository_LinkToAnalysis(t *testing.T) {
	ctx := context.Background()
	graph := setupTestGraph(t)
	repo := NewGraphExecutionPlanRepository(graph)

	// Create analysis node first (this would normally be done by the analysis repository)
	analysisID := "analysis-123"
	analysisData := map[string]interface{}{
		"id":          analysisID,
		"name":        "Test Analysis",
		"description": "Test analysis for linking",
	}
	err := graph.AddNode(ctx, "analysis", analysisID, analysisData)
	require.NoError(t, err)

	// Create test plan
	plan := domain.NewExecutionPlan("Test Plan", "Description", domain.ExecutionPlanPriorityMedium)
	err = repo.Create(ctx, plan)
	require.NoError(t, err)

	// Link to analysis
	err = repo.LinkToAnalysis(ctx, analysisID, plan.ID)
	require.NoError(t, err)

	// Verify link by retrieving plan by analysis ID
	retrievedPlan, err := repo.GetByAnalysisID(ctx, analysisID)
	require.NoError(t, err)
	assert.Equal(t, plan.ID, retrievedPlan.ID)
}

func TestGraphExecutionPlanRepository_Update(t *testing.T) {
	ctx := context.Background()
	graph := setupTestGraph(t)
	repo := NewGraphExecutionPlanRepository(graph)

	// Create test plan
	plan := domain.NewExecutionPlan("Original Name", "Description", domain.ExecutionPlanPriorityMedium)
	err := repo.Create(ctx, plan)
	require.NoError(t, err)

	// Update plan
	plan.Name = "Updated Name"
	plan.Approve()

	err = repo.Update(ctx, plan)
	require.NoError(t, err)

	// Verify update
	retrievedPlan, err := repo.GetByID(ctx, plan.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Name", retrievedPlan.Name)
	assert.Equal(t, domain.ExecutionPlanStatusApproved, retrievedPlan.Status)
	assert.NotNil(t, retrievedPlan.ApprovedAt)
}

func TestGraphExecutionPlanRepository_AddStep(t *testing.T) {
	ctx := context.Background()
	graph := setupTestGraph(t)
	repo := NewGraphExecutionPlanRepository(graph)

	// Create test plan
	plan := domain.NewExecutionPlan("Test Plan", "Description", domain.ExecutionPlanPriorityMedium)
	err := repo.Create(ctx, plan)
	require.NoError(t, err)

	// Add step
	step := domain.NewExecutionStep("New Step", "Step description", "agent-1")
	step.PlanID = plan.ID
	step.StepNumber = 1

	err = repo.AddStep(ctx, step)
	require.NoError(t, err)

	// Verify step was added
	steps, err := repo.GetStepsByPlanID(ctx, plan.ID)
	require.NoError(t, err)
	assert.Len(t, steps, 1)
	assert.Equal(t, step.ID, steps[0].ID)
	assert.Equal(t, step.Name, steps[0].Name)
}

func TestGraphExecutionPlanRepository_UpdateStep(t *testing.T) {
	ctx := context.Background()
	graph := setupTestGraph(t)
	repo := NewGraphExecutionPlanRepository(graph)

	// Create plan with step
	plan := domain.NewExecutionPlan("Test Plan", "Description", domain.ExecutionPlanPriorityMedium)
	step := domain.NewExecutionStep("Original Step", "Description", "agent-1")
	plan.AddStep(step)

	err := repo.Create(ctx, plan)
	require.NoError(t, err)

	// Update step
	step.Name = "Updated Step"
	step.Status = domain.ExecutionStepStatusCompleted
	step.Outputs = `{"result": "success"}`

	err = repo.UpdateStep(ctx, step)
	require.NoError(t, err)

	// Verify update
	steps, err := repo.GetStepsByPlanID(ctx, plan.ID)
	require.NoError(t, err)
	assert.Len(t, steps, 1)
	assert.Equal(t, "Updated Step", steps[0].Name)
	assert.Equal(t, domain.ExecutionStepStatusCompleted, steps[0].Status)
	assert.Equal(t, `{"result": "success"}`, steps[0].Outputs)
}

func TestGraphExecutionPlanRepository_AssignStepToAgent(t *testing.T) {
	ctx := context.Background()
	graph := setupTestGraph(t)
	repo := NewGraphExecutionPlanRepository(graph)

	// Create plan with step
	plan := domain.NewExecutionPlan("Test Plan", "Description", domain.ExecutionPlanPriorityMedium)
	step := domain.NewExecutionStep("Test Step", "Description", "old-agent")
	plan.AddStep(step)

	err := repo.Create(ctx, plan)
	require.NoError(t, err)

	// Reassign step to different agent
	newAgentID := "new-agent"
	err = repo.AssignStepToAgent(ctx, step.ID, newAgentID)
	require.NoError(t, err)

	// Verify reassignment
	steps, err := repo.GetStepsByPlanID(ctx, plan.ID)
	require.NoError(t, err)
	assert.Len(t, steps, 1)
	assert.Equal(t, newAgentID, steps[0].AssignedAgent)
}

func TestGraphExecutionPlanRepository_EnsureSchema(t *testing.T) {
	ctx := context.Background()
	graph := setupTestGraph(t)
	repo := NewGraphExecutionPlanRepository(graph)

	// Should not error when ensuring schema
	err := repo.EnsureSchema(ctx)
	assert.NoError(t, err)

	// Should be idempotent
	err = repo.EnsureSchema(ctx)
	assert.NoError(t, err)
}
