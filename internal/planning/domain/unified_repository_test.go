package domain

import (
	"context"
	"fmt"
	"testing"

	executionDomain "neuromesh/internal/execution/domain"
)

// TestUnifiedExecutionPlanRepository_TDD validates the consolidated repository interface
func TestUnifiedExecutionPlanRepository_TDD(t *testing.T) {
	t.Run("RED: should support unified operations that replace both PlanningResult and ExecutionPlan repositories", func(t *testing.T) {
		// This test validates that the unified repository can handle all operations
		// that were previously split between PlanningResultRepository and ExecutionPlanRepository

		// Create mock repository (this will fail until we update mocks)
		repo := &MockUnifiedExecutionPlanRepository{}

		// Test unified plan creation with both planning and execution data
		plan := NewUnifiedExecutionPlan(
			"plan-123",
			"Unified Test Plan",
			"Testing consolidated repository",
			ExecutionPlanPriorityHigh,
			"req-456",
			"test_unified_repo",
			"repository_testing",
			95,
			"Testing unified repository operations",
			[]string{"agent1", "agent2"},
			[]string{"agent1"},
			[]string{},
			PlanningTypeExecute,
		)

		// Should be able to create unified plan (combines planning result + execution plan creation)
		err := repo.Create(nil, plan)
		if err != nil {
			t.Errorf("Failed to create unified plan: %v", err)
		}

		// Should be able to retrieve by request ID (from former PlanningResultRepository)
		plans, err := repo.GetByRequestID(nil, "req-456")
		if err != nil {
			t.Errorf("Failed to get plans by request ID: %v", err)
		}
		if len(plans) == 0 {
			t.Error("Expected to find plans by request ID")
		}

		// Should be able to link to conversation (from former PlanningResultRepository)
		err = repo.LinkToConversation(nil, "plan-123", "conv-789")
		if err != nil {
			t.Errorf("Failed to link plan to conversation: %v", err)
		}

		// Should be able to delete plan (from former PlanningResultRepository)
		err = repo.Delete(nil, "plan-123")
		if err != nil {
			t.Errorf("Failed to delete unified plan: %v", err)
		}
	})

	t.Run("RED: should eliminate need for separate PlanningResultRepository", func(t *testing.T) {
		// This test documents that PlanningResultRepository is no longer needed
		// All its operations are now handled by unified ExecutionPlanRepository

		repo := &MockUnifiedExecutionPlanRepository{}

		// All planning result operations should work on ExecutionPlan
		plan := NewUnifiedExecutionPlan(
			"plan-consolidated",
			"No More PlanningResult",
			"All in ExecutionPlan",
			ExecutionPlanPriorityMedium,
			"req-consolidated",
			"eliminate_planning_result",
			"consolidation",
			100,
			"PlanningResult entity is no longer needed",
			[]string{"agent1"},
			[]string{"agent1"},
			[]string{},
			PlanningTypeExecute,
		)

		// Create unified plan
		err := repo.Create(nil, plan)
		if err != nil {
			t.Errorf("Failed to create consolidated plan: %v", err)
		}

		// Link to request (formerly PlanningResultRepository.LinkToRequest)
		err = repo.LinkToRequest(nil, "plan-consolidated", "req-consolidated")
		if err != nil {
			t.Errorf("Failed to link to request: %v", err)
		}

		// Get by request ID (formerly PlanningResultRepository.GetByRequestID)
		plans, err := repo.GetByRequestID(nil, "req-consolidated")
		if err != nil {
			t.Errorf("Failed to get by request ID: %v", err)
		}

		if len(plans) == 0 {
			t.Error("Should find plans by request ID")
		}

		// Validate plan contains both planning and execution data
		foundPlan := plans[0]
		if foundPlan.RequestID != "req-consolidated" {
			t.Error("Plan should contain planning data (RequestID)")
		}
		if foundPlan.Intent != "eliminate_planning_result" {
			t.Error("Plan should contain planning data (Intent)")
		}
		if foundPlan.Status != ExecutionPlanStatusDraft {
			t.Error("Plan should contain execution data (Status)")
		}
		if foundPlan.Priority != ExecutionPlanPriorityMedium {
			t.Error("Plan should contain execution data (Priority)")
		}
	})
}

// MockUnifiedExecutionPlanRepository is a mock implementation for testing
// This will fail to compile until we create it - that's our RED phase
type MockUnifiedExecutionPlanRepository struct {
	plans map[string]*ExecutionPlan
}

func (m *MockUnifiedExecutionPlanRepository) Create(ctx context.Context, plan *ExecutionPlan) error {
	if m.plans == nil {
		m.plans = make(map[string]*ExecutionPlan)
	}
	m.plans[plan.ID] = plan
	return nil
}

func (m *MockUnifiedExecutionPlanRepository) GetByID(ctx context.Context, id string) (*ExecutionPlan, error) {
	if plan, exists := m.plans[id]; exists {
		return plan, nil
	}
	return nil, fmt.Errorf("plan not found: %s", id)
}

func (m *MockUnifiedExecutionPlanRepository) GetByRequestID(ctx context.Context, requestID string) ([]*ExecutionPlan, error) {
	var plans []*ExecutionPlan
	for _, plan := range m.plans {
		if plan.RequestID == requestID {
			plans = append(plans, plan)
		}
	}
	return plans, nil
}

func (m *MockUnifiedExecutionPlanRepository) Delete(ctx context.Context, id string) error {
	delete(m.plans, id)
	return nil
}

func (m *MockUnifiedExecutionPlanRepository) LinkToConversation(ctx context.Context, planID, conversationID string) error {
	// Mock implementation - in real implementation this would create graph relationship
	return nil
}

func (m *MockUnifiedExecutionPlanRepository) LinkToRequest(ctx context.Context, planID, requestID string) error {
	// Mock implementation - in real implementation this would create graph relationship
	return nil
}

// Implement remaining required interface methods (minimal for test)
func (m *MockUnifiedExecutionPlanRepository) GetByAnalysisID(ctx context.Context, analysisID string) (*ExecutionPlan, error) {
	return nil, nil
}
func (m *MockUnifiedExecutionPlanRepository) Update(ctx context.Context, plan *ExecutionPlan) error {
	return nil
}
func (m *MockUnifiedExecutionPlanRepository) LinkToAnalysis(ctx context.Context, analysisID, planID string) error {
	return nil
}
func (m *MockUnifiedExecutionPlanRepository) GetStepsByPlanID(ctx context.Context, planID string) ([]*ExecutionStep, error) {
	return nil, nil
}
func (m *MockUnifiedExecutionPlanRepository) AddStep(ctx context.Context, step *ExecutionStep) error {
	return nil
}
func (m *MockUnifiedExecutionPlanRepository) UpdateStep(ctx context.Context, step *ExecutionStep) error {
	return nil
}
func (m *MockUnifiedExecutionPlanRepository) AssignStepToAgent(ctx context.Context, stepID, agentID string) error {
	return nil
}
func (m *MockUnifiedExecutionPlanRepository) StoreAgentResult(ctx context.Context, result *executionDomain.AgentResult) error {
	return nil
}
func (m *MockUnifiedExecutionPlanRepository) GetAgentResultsByExecutionPlan(ctx context.Context, planID string) ([]*executionDomain.AgentResult, error) {
	return nil, nil
}
func (m *MockUnifiedExecutionPlanRepository) GetAgentResultsByExecutionStep(ctx context.Context, stepID string) ([]*executionDomain.AgentResult, error) {
	return nil, nil
}
func (m *MockUnifiedExecutionPlanRepository) GetAgentResultByID(ctx context.Context, resultID string) (*executionDomain.AgentResult, error) {
	return nil, nil
}
func (m *MockUnifiedExecutionPlanRepository) GetPlanIDByCorrelationID(ctx context.Context, correlationID string) (string, error) {
	return "", nil
}
func (m *MockUnifiedExecutionPlanRepository) StoreSynthesisResult(ctx context.Context, result *executionDomain.SynthesisResult) error {
	return nil
}
func (m *MockUnifiedExecutionPlanRepository) GetSynthesisResultByPlanID(ctx context.Context, planID string) (*executionDomain.SynthesisResult, error) {
	return nil, nil
}
