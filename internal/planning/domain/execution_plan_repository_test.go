package domain

import (
	"context"
	"testing"

	executionDomain "neuromesh/internal/execution/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockExecutionPlanRepository is a mock implementation for testing
type MockExecutionPlanRepository struct {
	mock.Mock
}

func (m *MockExecutionPlanRepository) Create(ctx context.Context, plan *ExecutionPlan) error {
	args := m.Called(ctx, plan)
	return args.Error(0)
}

func (m *MockExecutionPlanRepository) GetByID(ctx context.Context, id string) (*ExecutionPlan, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ExecutionPlan), args.Error(1)
}

func (m *MockExecutionPlanRepository) GetByAnalysisID(ctx context.Context, analysisID string) (*ExecutionPlan, error) {
	args := m.Called(ctx, analysisID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ExecutionPlan), args.Error(1)
}

func (m *MockExecutionPlanRepository) Update(ctx context.Context, plan *ExecutionPlan) error {
	args := m.Called(ctx, plan)
	return args.Error(0)
}

func (m *MockExecutionPlanRepository) LinkToAnalysis(ctx context.Context, analysisID, planID string) error {
	args := m.Called(ctx, analysisID, planID)
	return args.Error(0)
}

func (m *MockExecutionPlanRepository) GetStepsByPlanID(ctx context.Context, planID string) ([]*ExecutionStep, error) {
	args := m.Called(ctx, planID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*ExecutionStep), args.Error(1)
}

func (m *MockExecutionPlanRepository) AddStep(ctx context.Context, step *ExecutionStep) error {
	args := m.Called(ctx, step)
	return args.Error(0)
}

func (m *MockExecutionPlanRepository) UpdateStep(ctx context.Context, step *ExecutionStep) error {
	args := m.Called(ctx, step)
	return args.Error(0)
}

func (m *MockExecutionPlanRepository) AssignStepToAgent(ctx context.Context, stepID, agentID string) error {
	args := m.Called(ctx, stepID, agentID)
	return args.Error(0)
}

// Agent Result operations - Mock implementations for the new interface methods
func (m *MockExecutionPlanRepository) StoreAgentResult(ctx context.Context, result *executionDomain.AgentResult) error {
	args := m.Called(ctx, result)
	return args.Error(0)
}

func (m *MockExecutionPlanRepository) GetAgentResultByID(ctx context.Context, resultID string) (*executionDomain.AgentResult, error) {
	args := m.Called(ctx, resultID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*executionDomain.AgentResult), args.Error(1)
}

func (m *MockExecutionPlanRepository) GetAgentResultsByExecutionStep(ctx context.Context, stepID string) ([]*executionDomain.AgentResult, error) {
	args := m.Called(ctx, stepID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*executionDomain.AgentResult), args.Error(1)
}

func (m *MockExecutionPlanRepository) GetAgentResultsByExecutionPlan(ctx context.Context, planID string) ([]*executionDomain.AgentResult, error) {
	args := m.Called(ctx, planID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*executionDomain.AgentResult), args.Error(1)
}

func TestExecutionPlanRepository_Interface(t *testing.T) {
	// This test ensures our mock implements the interface correctly
	var repo ExecutionPlanRepository = &MockExecutionPlanRepository{}
	assert.NotNil(t, repo)
}

func TestMockExecutionPlanRepository_Create(t *testing.T) {
	mockRepo := &MockExecutionPlanRepository{}
	plan := &ExecutionPlan{
		ID:   "plan-123",
		Name: "Test Plan",
	}
	ctx := context.Background()

	mockRepo.On("Create", ctx, plan).Return(nil)

	err := mockRepo.Create(ctx, plan)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestMockExecutionPlanRepository_GetByID(t *testing.T) {
	mockRepo := &MockExecutionPlanRepository{}
	expectedPlan := &ExecutionPlan{
		ID:   "plan-123",
		Name: "Test Plan",
	}
	ctx := context.Background()

	mockRepo.On("GetByID", ctx, "plan-123").Return(expectedPlan, nil)

	plan, err := mockRepo.GetByID(ctx, "plan-123")
	assert.NoError(t, err)
	assert.Equal(t, expectedPlan, plan)
	mockRepo.AssertExpectations(t)
}

func TestMockExecutionPlanRepository_LinkToAnalysis(t *testing.T) {
	mockRepo := &MockExecutionPlanRepository{}
	ctx := context.Background()

	mockRepo.On("LinkToAnalysis", ctx, "analysis-123", "plan-123").Return(nil)

	err := mockRepo.LinkToAnalysis(ctx, "analysis-123", "plan-123")
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}
