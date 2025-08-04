package testHelpers

import (
	"context"
	"fmt"
	"sync"

	executionDomain "neuromesh/internal/execution/domain"
	"neuromesh/internal/planning/domain"
)

// MockExecutionPlanRepository is a mock implementation of ExecutionPlanRepository for testing
type MockExecutionPlanRepository struct {
	mu            sync.RWMutex
	plans         map[string]*domain.ExecutionPlan
	steps         map[string][]*domain.ExecutionStep
	analysisLinks map[string]string                       // analysisID -> planID
	agentResults  map[string]*executionDomain.AgentResult // resultID -> AgentResult
	calls         []string
}

// NewMockExecutionPlanRepository creates a new mock execution plan repository
func NewMockExecutionPlanRepository() *MockExecutionPlanRepository {
	return &MockExecutionPlanRepository{
		plans:         make(map[string]*domain.ExecutionPlan),
		steps:         make(map[string][]*domain.ExecutionStep),
		analysisLinks: make(map[string]string),
		agentResults:  make(map[string]*executionDomain.AgentResult),
		calls:         make([]string, 0),
	}
}

// Create stores a new execution plan
func (m *MockExecutionPlanRepository) Create(ctx context.Context, plan *domain.ExecutionPlan) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.calls = append(m.calls, fmt.Sprintf("Create(%s)", plan.ID))
	m.plans[plan.ID] = plan

	// Store steps separately
	if len(plan.Steps) > 0 {
		m.steps[plan.ID] = make([]*domain.ExecutionStep, len(plan.Steps))
		copy(m.steps[plan.ID], plan.Steps)
	}

	return nil
}

// GetByID retrieves an execution plan by ID
func (m *MockExecutionPlanRepository) GetByID(ctx context.Context, id string) (*domain.ExecutionPlan, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	m.calls = append(m.calls, fmt.Sprintf("GetByID(%s)", id))

	plan, exists := m.plans[id]
	if !exists {
		return nil, fmt.Errorf("execution plan not found: %s", id)
	}

	// Load steps
	if steps, hasSteps := m.steps[id]; hasSteps {
		plan.Steps = make([]*domain.ExecutionStep, len(steps))
		copy(plan.Steps, steps)
	}

	return plan, nil
}

// GetByAnalysisID retrieves an execution plan by analysis ID
func (m *MockExecutionPlanRepository) GetByAnalysisID(ctx context.Context, analysisID string) (*domain.ExecutionPlan, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	m.calls = append(m.calls, fmt.Sprintf("GetByAnalysisID(%s)", analysisID))

	planID, exists := m.analysisLinks[analysisID]
	if !exists {
		return nil, fmt.Errorf("no execution plan found for analysis: %s", analysisID)
	}

	return m.GetByID(ctx, planID)
}

// Update updates an execution plan
func (m *MockExecutionPlanRepository) Update(ctx context.Context, plan *domain.ExecutionPlan) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.calls = append(m.calls, fmt.Sprintf("Update(%s)", plan.ID))

	if _, exists := m.plans[plan.ID]; !exists {
		return fmt.Errorf("execution plan not found: %s", plan.ID)
	}

	m.plans[plan.ID] = plan
	return nil
}

// LinkToAnalysis links an execution plan to an analysis
func (m *MockExecutionPlanRepository) LinkToAnalysis(ctx context.Context, analysisID, planID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.calls = append(m.calls, fmt.Sprintf("LinkToAnalysis(%s, %s)", analysisID, planID))
	m.analysisLinks[analysisID] = planID
	return nil
}

// GetStepsByPlanID retrieves all steps for a plan
func (m *MockExecutionPlanRepository) GetStepsByPlanID(ctx context.Context, planID string) ([]*domain.ExecutionStep, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	m.calls = append(m.calls, fmt.Sprintf("GetStepsByPlanID(%s)", planID))

	steps, exists := m.steps[planID]
	if !exists {
		return []*domain.ExecutionStep{}, nil
	}

	result := make([]*domain.ExecutionStep, len(steps))
	copy(result, steps)
	return result, nil
}

// AddStep adds a step to a plan
func (m *MockExecutionPlanRepository) AddStep(ctx context.Context, step *domain.ExecutionStep) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.calls = append(m.calls, fmt.Sprintf("AddStep(%s)", step.ID))

	if step.PlanID != "" {
		if _, exists := m.steps[step.PlanID]; !exists {
			m.steps[step.PlanID] = make([]*domain.ExecutionStep, 0)
		}
		m.steps[step.PlanID] = append(m.steps[step.PlanID], step)
	}

	return nil
}

// UpdateStep updates a step
func (m *MockExecutionPlanRepository) UpdateStep(ctx context.Context, step *domain.ExecutionStep) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.calls = append(m.calls, fmt.Sprintf("UpdateStep(%s)", step.ID))

	// Find and update the step
	if steps, exists := m.steps[step.PlanID]; exists {
		for i, s := range steps {
			if s.ID == step.ID {
				m.steps[step.PlanID][i] = step
				return nil
			}
		}
	}

	return fmt.Errorf("step not found: %s", step.ID)
}

// AssignStepToAgent assigns a step to an agent
func (m *MockExecutionPlanRepository) AssignStepToAgent(ctx context.Context, stepID, agentID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.calls = append(m.calls, fmt.Sprintf("AssignStepToAgent(%s, %s)", stepID, agentID))

	// Find and update the step's agent assignment
	for _, steps := range m.steps {
		for _, step := range steps {
			if step.ID == stepID {
				step.AssignedAgent = agentID
				return nil
			}
		}
	}

	return fmt.Errorf("step not found: %s", stepID)
}

// GetCalls returns all method calls made to this mock (for testing)
func (m *MockExecutionPlanRepository) GetCalls() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]string, len(m.calls))
	copy(result, m.calls)
	return result
}

// GetPlanCount returns the number of plans stored
func (m *MockExecutionPlanRepository) GetPlanCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.plans)
}

// GetLinkCount returns the number of analysis links
func (m *MockExecutionPlanRepository) GetLinkCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.analysisLinks)
}

// Agent Result operations - NEW for graph-native result synthesis

// StoreAgentResult stores an agent result in the mock
func (m *MockExecutionPlanRepository) StoreAgentResult(ctx context.Context, result *executionDomain.AgentResult) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.calls = append(m.calls, fmt.Sprintf("StoreAgentResult(%s)", result.ID))
	m.agentResults[result.ID] = result
	return nil
}

// GetAgentResultByID retrieves a specific agent result by its ID
func (m *MockExecutionPlanRepository) GetAgentResultByID(ctx context.Context, resultID string) (*executionDomain.AgentResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.calls = append(m.calls, fmt.Sprintf("GetAgentResultByID(%s)", resultID))

	result, exists := m.agentResults[resultID]
	if !exists {
		return nil, fmt.Errorf("agent result %s not found", resultID)
	}

	return result, nil
}

// GetAgentResultsByExecutionStep retrieves all agent results for a specific execution step
func (m *MockExecutionPlanRepository) GetAgentResultsByExecutionStep(ctx context.Context, stepID string) ([]*executionDomain.AgentResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.calls = append(m.calls, fmt.Sprintf("GetAgentResultsByExecutionStep(%s)", stepID))

	var results []*executionDomain.AgentResult
	for _, result := range m.agentResults {
		if result.ExecutionStepID == stepID {
			results = append(results, result)
		}
	}

	return results, nil
}

// GetAgentResultsByExecutionPlan retrieves all agent results for an entire execution plan
func (m *MockExecutionPlanRepository) GetAgentResultsByExecutionPlan(ctx context.Context, planID string) ([]*executionDomain.AgentResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.calls = append(m.calls, fmt.Sprintf("GetAgentResultsByExecutionPlan(%s)", planID))

	// First get all steps for the plan
	planSteps, exists := m.steps[planID]
	if !exists {
		return []*executionDomain.AgentResult{}, nil
	}

	// Collect all results for all steps in the plan
	var results []*executionDomain.AgentResult
	for _, step := range planSteps {
		for _, result := range m.agentResults {
			if result.ExecutionStepID == step.ID {
				results = append(results, result)
			}
		}
	}

	return results, nil
}

// GetStoredAgentResults returns all stored agent results for testing
func (m *MockExecutionPlanRepository) GetStoredAgentResults() []*executionDomain.AgentResult {
	m.mu.RLock()
	defer m.mu.RUnlock()

	results := make([]*executionDomain.AgentResult, 0, len(m.agentResults))
	for _, result := range m.agentResults {
		results = append(results, result)
	}
	return results
}

// GetPlanIDByCorrelationID finds the execution plan ID for a given correlation ID
func (m *MockExecutionPlanRepository) GetPlanIDByCorrelationID(ctx context.Context, correlationID string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	m.calls = append(m.calls, fmt.Sprintf("GetPlanIDByCorrelationID(%s)", correlationID))

	// In the mock, we'll look through steps to find the plan
	for planID, steps := range m.steps {
		for _, step := range steps {
			if step.ID == correlationID {
				return planID, nil
			}
		}
	}

	// Not found
	return "", nil
}

// StoreConversationSummary stores a conversation summary (mock implementation)
func (m *MockExecutionPlanRepository) StoreConversationSummary(ctx context.Context, summary *executionDomain.ConversationSummary) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, fmt.Sprintf("StoreConversationSummary(%s)", summary.PlanID))
	// In a real mock we might store this, but for now just record the call
	return nil
}

// GetConversationSummaryByPlanID retrieves a conversation summary by plan ID (mock implementation)
func (m *MockExecutionPlanRepository) GetConversationSummaryByPlanID(ctx context.Context, planID string) (*executionDomain.ConversationSummary, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	m.calls = append(m.calls, fmt.Sprintf("GetConversationSummaryByPlanID(%s)", planID))
	// Mock returns nil - no conversation summary found
	return nil, nil
}

// AssertStoreConversationSummaryCalled verifies that StoreConversationSummary was called for the given planID
func (m *MockExecutionPlanRepository) AssertStoreConversationSummaryCalled(t interface {
	Errorf(format string, args ...interface{})
}, planID string) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	expectedCall := fmt.Sprintf("StoreConversationSummary(%s)", planID)
	for _, call := range m.calls {
		if call == expectedCall {
			return // Found the call, assertion passes
		}
	}

	// Call not found, assertion fails
	t.Errorf("Expected StoreConversationSummary to be called with planID '%s', but it was not. Recorded calls: %v", planID, m.calls)
}

// Delete removes an execution plan (unified repository method)
func (m *MockExecutionPlanRepository) Delete(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.calls = append(m.calls, fmt.Sprintf("Delete(%s)", id))
	delete(m.plans, id)
	delete(m.steps, id)
	return nil
}

// GetByRequestID retrieves execution plans by request ID (unified repository method)
func (m *MockExecutionPlanRepository) GetByRequestID(ctx context.Context, requestID string) ([]*domain.ExecutionPlan, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	m.calls = append(m.calls, fmt.Sprintf("GetByRequestID(%s)", requestID))

	var plans []*domain.ExecutionPlan
	for _, plan := range m.plans {
		if plan.RequestID == requestID {
			plans = append(plans, plan)
		}
	}

	return plans, nil
}

// LinkToRequest links execution plan to a request (unified repository method)
func (m *MockExecutionPlanRepository) LinkToRequest(ctx context.Context, planID, requestID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.calls = append(m.calls, fmt.Sprintf("LinkToRequest(%s, %s)", planID, requestID))
	if plan, exists := m.plans[planID]; exists {
		plan.RequestID = requestID
	}
	return nil
}

// LinkToConversation is deprecated - use Conversation domain's LinkExecutionPlan instead
// This mock now does nothing to reflect the architectural decision
func (m *MockExecutionPlanRepository) LinkToConversation(ctx context.Context, planID, conversationID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.calls = append(m.calls, fmt.Sprintf("LinkToConversation(%s, %s)", planID, conversationID))
	// No-op: Conversation domain owns the relationship
	return nil
}

// GetConversationIDByPlanID gets conversation ID linked to execution plan
func (m *MockExecutionPlanRepository) GetConversationIDByPlanID(ctx context.Context, planID string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.calls = append(m.calls, fmt.Sprintf("GetConversationIDByPlanID(%s)", planID))
	// Mock implementation - return mock conversation ID for testing
	return "mock-conversation-id", nil
}
