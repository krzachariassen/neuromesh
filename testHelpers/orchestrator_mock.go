package testHelpers

import (
	"context"

	"neuromesh/internal/execution/domain"

	"github.com/stretchr/testify/mock"
)

// MockGraphExplorer provides a testify-based mock for graph explorer operations
type MockGraphExplorer struct {
	mock.Mock
}

// NewMockGraphExplorer creates a new mock graph explorer instance
func NewMockGraphExplorer() *MockGraphExplorer {
	return &MockGraphExplorer{}
}

// NOTE: Legacy Analysis/Decision methods removed - following YAGNI principles
// These mocks were part of the old dual-step orchestration pattern
// New unified planning approach uses PlanningEngine directly

// NOTE: MockExecutionCoordinator and MockLearningService removed - following YAGNI principles
// Legacy coordination patterns replaced by new unified planning approach

// MockExecutionService provides a testify-based mock for execution service operations
type MockExecutionService struct {
	mock.Mock
}

// NewMockExecutionService creates a new mock execution service instance
func NewMockExecutionService() *MockExecutionService {
	return &MockExecutionService{}
}

func (m *MockExecutionService) CreateExecutionPlan(ctx context.Context, plan *domain.ExecutionPlan) error {
	args := m.Called(ctx, plan)
	return args.Error(0)
}

func (m *MockExecutionService) GetExecutionPlan(ctx context.Context, planID string) (*domain.ExecutionPlan, error) {
	args := m.Called(ctx, planID)
	return args.Get(0).(*domain.ExecutionPlan), args.Error(1)
}

func (m *MockExecutionService) UpdateExecutionStatus(ctx context.Context, planID string, status domain.ExecutionStatus) error {
	args := m.Called(ctx, planID, status)
	return args.Error(0)
}
