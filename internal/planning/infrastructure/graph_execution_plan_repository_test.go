package infrastructure

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	executionDomain "neuromesh/internal/execution/domain"
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

// RED Phase: Tests for AgentResult graph storage - these will fail until implemented
func TestGraphExecutionPlanRepository_StoreAgentResult_ShouldPersistToGraph(t *testing.T) {
	ctx := context.Background()
	graph := setupTestGraph(t)
	repo := NewGraphExecutionPlanRepository(graph)

	// Create a plan with a step first
	plan := domain.NewExecutionPlan("Test Plan", "Test description", domain.ExecutionPlanPriorityMedium)
	step := domain.NewExecutionStep("Test Step", "Test step description", "test-agent-123")
	plan.AddStep(step)

	err := repo.Create(ctx, plan)
	require.NoError(t, err)

	// Create an agent result
	metadata := map[string]interface{}{
		"execution_time": 2.5,
		"confidence":     0.95,
	}
	result := executionDomain.NewAgentResult(step.ID, "test-agent-123", "Diagnostic analysis complete", metadata)

	// Act: Store the agent result
	err = repo.StoreAgentResult(ctx, result)

	// Assert: Should store without error
	require.NoError(t, err, "StoreAgentResult should persist agent result to graph")

	// Verify result can be retrieved by ID
	retrievedResult, err := repo.GetAgentResultByID(ctx, result.ID)
	require.NoError(t, err, "Should be able to retrieve stored agent result")
	assert.Equal(t, result.ID, retrievedResult.ID)
	assert.Equal(t, result.ExecutionStepID, retrievedResult.ExecutionStepID)
	assert.Equal(t, result.AgentID, retrievedResult.AgentID)
	assert.Equal(t, result.Content, retrievedResult.Content)
	assert.Equal(t, result.Status, retrievedResult.Status)
}

func TestGraphExecutionPlanRepository_GetAgentResultsByExecutionStep_ShouldReturnResultsForStep(t *testing.T) {
	ctx := context.Background()
	graph := setupTestGraph(t)
	repo := NewGraphExecutionPlanRepository(graph)

	// Create plan with multiple steps
	plan := domain.NewExecutionPlan("Multi-Agent Plan", "Test plan with multiple agents", domain.ExecutionPlanPriorityHigh)
	step1 := domain.NewExecutionStep("Symptom Analysis", "Analyze patient symptoms", "symptom-agent")
	step2 := domain.NewExecutionStep("Diagnostic Analysis", "Perform diagnostic analysis", "diagnostic-agent")
	plan.AddStep(step1)
	plan.AddStep(step2)

	err := repo.Create(ctx, plan)
	require.NoError(t, err)

	// Create multiple results for step1
	result1 := executionDomain.NewAgentResult(step1.ID, "symptom-agent", "Initial symptom analysis", nil)
	result2 := executionDomain.NewAgentResultWithStatus(step1.ID, "symptom-agent", "Refined analysis", nil, executionDomain.AgentResultStatusPartial)

	// Create one result for step2
	result3 := executionDomain.NewAgentResult(step2.ID, "diagnostic-agent", "Diagnostic complete", nil)

	err = repo.StoreAgentResult(ctx, result1)
	require.NoError(t, err)
	err = repo.StoreAgentResult(ctx, result2)
	require.NoError(t, err)
	err = repo.StoreAgentResult(ctx, result3)
	require.NoError(t, err)

	// Act: Get results for step1 only
	step1Results, err := repo.GetAgentResultsByExecutionStep(ctx, step1.ID)

	// Assert: Should return only step1 results
	require.NoError(t, err)
	assert.Len(t, step1Results, 2, "Should return exactly 2 results for step1")

	// Verify all results belong to step1
	for _, result := range step1Results {
		assert.Equal(t, step1.ID, result.ExecutionStepID)
	}
}

func TestGraphExecutionPlanRepository_GetAgentResultsByExecutionPlan_ShouldReturnAllPlanResults(t *testing.T) {
	ctx := context.Background()
	graph := setupTestGraph(t)
	repo := NewGraphExecutionPlanRepository(graph)

	// Create plan with multiple steps
	plan := domain.NewExecutionPlan("Healthcare Diagnosis", "Multi-agent diagnostic plan", domain.ExecutionPlanPriorityHigh)
	step1 := domain.NewExecutionStep("Symptom Analysis", "Analyze symptoms", "symptom-agent")
	step2 := domain.NewExecutionStep("Lab Analysis", "Analyze lab results", "lab-agent")
	step3 := domain.NewExecutionStep("Diagnosis", "Create diagnosis", "diagnostic-agent")
	plan.AddStep(step1)
	plan.AddStep(step2)
	plan.AddStep(step3)

	err := repo.Create(ctx, plan)
	require.NoError(t, err)

	// Create results for all steps
	result1 := executionDomain.NewAgentResult(step1.ID, "symptom-agent", "Symptom analysis: chest pain, dyspnea", nil)
	result2 := executionDomain.NewAgentResult(step2.ID, "lab-agent", "Lab results: elevated troponin", nil)
	result3 := executionDomain.NewAgentResult(step3.ID, "diagnostic-agent", "Diagnosis: Acute myocardial infarction", nil)

	err = repo.StoreAgentResult(ctx, result1)
	require.NoError(t, err)
	err = repo.StoreAgentResult(ctx, result2)
	require.NoError(t, err)
	err = repo.StoreAgentResult(ctx, result3)
	require.NoError(t, err)

	// Act: Get all results for the plan
	planResults, err := repo.GetAgentResultsByExecutionPlan(ctx, plan.ID)

	// Assert: Should return all results across all steps
	require.NoError(t, err)
	assert.Len(t, planResults, 3, "Should return all 3 results from the execution plan")

	// Verify results are from correct steps
	stepIDs := []string{step1.ID, step2.ID, step3.ID}
	resultStepIDs := make([]string, len(planResults))
	for i, result := range planResults {
		resultStepIDs[i] = result.ExecutionStepID
	}

	for _, stepID := range stepIDs {
		assert.Contains(t, resultStepIDs, stepID, "Should contain result from step %s", stepID)
	}
}

func TestGraphExecutionPlanRepository_StoreAgentResult_ValidationError_ShouldReturnError(t *testing.T) {
	ctx := context.Background()
	graph := setupTestGraph(t)
	repo := NewGraphExecutionPlanRepository(graph)

	// Create invalid agent result (missing required fields)
	invalidResult := &executionDomain.AgentResult{
		ID:      "", // Missing ID
		Content: "Some content",
	}

	// Act: Try to store invalid result
	err := repo.StoreAgentResult(ctx, invalidResult)

	// Assert: Should return validation error
	require.Error(t, err, "Should return error for invalid agent result")
	assert.Contains(t, err.Error(), "validation", "Error should mention validation")
}

func TestGraphExecutionPlanRepository_GetAgentResultByID_NotFound_ShouldReturnError(t *testing.T) {
	ctx := context.Background()
	graph := setupTestGraph(t)
	repo := NewGraphExecutionPlanRepository(graph)

	// Act: Try to get non-existent result
	result, err := repo.GetAgentResultByID(ctx, "non-existent-result-id")

	// Assert: Should return error and nil result
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "not found", "Error should indicate result was not found")
}
