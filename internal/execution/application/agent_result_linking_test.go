package application

import (
	"context"
	"errors"
	"testing"

	executionDomain "neuromesh/internal/execution/domain"
	"neuromesh/internal/messaging"
	planningDomain "neuromesh/internal/planning/domain"
)

// TestAgentResultLinkingToExecutionPlan_TDD tests the critical flow that's failing in production
func TestAgentResultLinkingToExecutionPlan_TDD(t *testing.T) {
	ctx := context.Background()

	t.Run("RED: agent result should be linked to execution plan step", func(t *testing.T) {
		// This test exposes the exact issue we found in the real platform test:
		// Agent results are not being properly linked to execution plan steps

		// Arrange: Create a real execution plan with proper execution step ID
		planID := "test-plan-123"
		executionStepID := "a1b2c3d4-e5f6-7890-abcd-ef1234567890" // Proper UUID execution step ID

		// Create a mock repository that simulates the graph storage
		mockRepo := &TestExecutionPlanRepository{
			correlationToPlanMap: map[string]string{
				executionStepID: planID, // Map execution step ID to plan ID
			},
			plans:         make(map[string]*planningDomain.ExecutionPlan),
			steps:         make(map[string][]*planningDomain.ExecutionStep),
			storedResults: make([]*executionDomain.AgentResult, 0),
		}

		engine := &AIExecutionEngine{
			repository: mockRepo,
		}

		// Create an execution plan with a step using proper execution step ID
		plan := &planningDomain.ExecutionPlan{
			ID:     planID,
			Status: planningDomain.ExecutionPlanStatusExecuting,
		}
		step := &planningDomain.ExecutionStep{
			ID:     executionStepID, // Execution step ID matches what agents receive
			PlanID: planID,
			Status: planningDomain.ExecutionStepStatusPending,
		}
		mockRepo.plans[planID] = plan
		mockRepo.steps[planID] = []*planningDomain.ExecutionStep{step}

		// Act: Simulate agent returning a result (like text-processor agent did)
		agentResponse := &messaging.AgentToAIMessage{
			CorrelationID: executionStepID, // Agent uses execution step ID (not session correlation)
			AgentID:       "text-processor-001",
			Content:       "The text contains 7 words",
			Context: map[string]interface{}{
				"step_id": executionStepID, // Include step ID in context for clarity
				"plan_id": planID,          // Include plan ID in context
			},
		}

		// This should store the agent result and link it to the execution plan
		err := engine.storeAgentResult(ctx, agentResponse)

		// Assert: The agent result should be properly linked
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		// Verify the plan ID was extracted correctly
		if len(mockRepo.storedResults) == 0 {
			t.Fatal("Expected agent result to be stored")
		}

		storedResult := mockRepo.storedResults[0]

		// The critical assertion: Agent result should contain plan ID metadata
		planIDFromMeta, exists := storedResult.Metadata["plan_id"].(string)
		if !exists {
			t.Error("Agent result metadata should contain plan_id")
		}
		if planIDFromMeta != planID {
			t.Errorf("Expected plan_id %s, got %s", planID, planIDFromMeta)
		}

		// The step should be marked as completed - get fresh copy from repository
		updatedSteps, err := mockRepo.GetStepsByPlanID(ctx, planID)
		if err != nil {
			t.Fatalf("Failed to get updated steps: %v", err)
		}
		if len(updatedSteps) == 0 {
			t.Fatal("Expected at least one step in plan")
		}

		updatedStep := updatedSteps[0]
		if updatedStep.Status != planningDomain.ExecutionStepStatusCompleted {
			t.Errorf("Expected step status to be completed, got %v", updatedStep.Status)
		}
	})
}

// TestExecutionPlanRepository implements the repository interface for testing
type TestExecutionPlanRepository struct {
	correlationToPlanMap map[string]string
	plans                map[string]*planningDomain.ExecutionPlan
	steps                map[string][]*planningDomain.ExecutionStep
	storedResults        []*executionDomain.AgentResult
}

func (m *TestExecutionPlanRepository) StoreAgentResult(ctx context.Context, result *executionDomain.AgentResult) error {
	m.storedResults = append(m.storedResults, result)
	return nil
}

func (m *TestExecutionPlanRepository) GetStepsByPlanID(ctx context.Context, planID string) ([]*planningDomain.ExecutionStep, error) {
	return m.steps[planID], nil
}

func (m *TestExecutionPlanRepository) UpdateStep(ctx context.Context, step *planningDomain.ExecutionStep) error {
	// Find the step and update it
	for _, steps := range m.steps {
		for i, s := range steps {
			if s.ID == step.ID {
				steps[i] = step
				return nil
			}
		}
	}
	return nil
}

// GetPlanIDByCorrelationID simulates graph lookup of correlation to plan mapping
func (m *TestExecutionPlanRepository) GetPlanIDByCorrelationID(ctx context.Context, correlationID string) (string, error) {
	planID, exists := m.correlationToPlanMap[correlationID]
	if !exists {
		return "", errors.New("plan not found for correlation ID: " + correlationID)
	}
	return planID, nil
}

// Implement other required methods as no-ops for now
func (m *TestExecutionPlanRepository) Create(ctx context.Context, plan *planningDomain.ExecutionPlan) error {
	return nil
}
func (m *TestExecutionPlanRepository) GetByID(ctx context.Context, id string) (*planningDomain.ExecutionPlan, error) {
	return m.plans[id], nil
}
func (m *TestExecutionPlanRepository) GetByAnalysisID(ctx context.Context, analysisID string) (*planningDomain.ExecutionPlan, error) {
	return nil, nil
}
func (m *TestExecutionPlanRepository) Update(ctx context.Context, plan *planningDomain.ExecutionPlan) error {
	return nil
}
func (m *TestExecutionPlanRepository) LinkToAnalysis(ctx context.Context, analysisID, planID string) error {
	return nil
}
func (m *TestExecutionPlanRepository) AddStep(ctx context.Context, step *planningDomain.ExecutionStep) error {
	return nil
}
func (m *TestExecutionPlanRepository) AssignStepToAgent(ctx context.Context, stepID, agentID string) error {
	return nil
}
func (m *TestExecutionPlanRepository) GetAgentResultsByExecutionPlan(ctx context.Context, planID string) ([]*executionDomain.AgentResult, error) {
	return nil, nil
}
func (m *TestExecutionPlanRepository) GetAgentResultsByExecutionStep(ctx context.Context, stepID string) ([]*executionDomain.AgentResult, error) {
	return nil, nil
}
func (m *TestExecutionPlanRepository) GetAgentResultByID(ctx context.Context, resultID string) (*executionDomain.AgentResult, error) {
	return nil, nil
}
func (m *TestExecutionPlanRepository) StoreSynthesisResult(ctx context.Context, result *executionDomain.SynthesisResult) error {
	return nil
}
func (m *TestExecutionPlanRepository) GetSynthesisResultByPlanID(ctx context.Context, planID string) (*executionDomain.SynthesisResult, error) {
	return nil, nil
}
