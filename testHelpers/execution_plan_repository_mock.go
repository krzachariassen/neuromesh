package testHelpers

import (
	"context"
	"fmt"
	"sync"

	"neuromesh/internal/planning/domain"
)

// MockExecutionPlanRepository is a mock implementation of ExecutionPlanRepository for testing
type MockExecutionPlanRepository struct {
	mu            sync.RWMutex
	plans         map[string]*domain.ExecutionPlan
	steps         map[string][]*domain.ExecutionStep
	analysisLinks map[string]string // analysisID -> planID
	calls         []string
}

// NewMockExecutionPlanRepository creates a new mock execution plan repository
func NewMockExecutionPlanRepository() *MockExecutionPlanRepository {
	return &MockExecutionPlanRepository{
		plans:         make(map[string]*domain.ExecutionPlan),
		steps:         make(map[string][]*domain.ExecutionStep),
		analysisLinks: make(map[string]string),
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
