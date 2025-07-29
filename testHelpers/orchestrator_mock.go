package testHelpers

import (
	"context"

	"neuromesh/internal/execution/domain"
	planningDomain "neuromesh/internal/planning/domain"

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

// GetAgentContext mocks the GetAgentContext method
func (m *MockGraphExplorer) GetAgentContext(ctx context.Context) (string, error) {
	args := m.Called(ctx)
	return args.String(0), args.Error(1)
}

// SetGetAgentContextResult sets up mock result for GetAgentContext
func (m *MockGraphExplorer) SetGetAgentContextResult(result string, err error) {
	m.On("GetAgentContext", mock.Anything).Return(result, err)
}

// MockAIPlanningEngine provides a testify-based mock for AI planning engine
type MockAIPlanningEngine struct {
	mock.Mock
}

// NewMockAIPlanningEngine creates a new mock AI planning engine instance
func NewMockAIPlanningEngine() *MockAIPlanningEngine {
	return &MockAIPlanningEngine{}
}

// CreateExecutionPlan mocks the CreateExecutionPlan method
func (m *MockAIPlanningEngine) CreateExecutionPlan(ctx context.Context, userInput, userID, agentContext, requestID string) (*planningDomain.PlanningResult, error) {
	args := m.Called(ctx, userInput, userID, agentContext, requestID)
	return args.Get(0).(*planningDomain.PlanningResult), args.Error(1)
}

// SetCreateExecutionPlanResult sets up mock result for CreateExecutionPlan
func (m *MockAIPlanningEngine) SetCreateExecutionPlanResult(result *planningDomain.PlanningResult, err error) {
	m.On("CreateExecutionPlan", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(result, err)
}

// LinkPlanningResultToConversation mocks the LinkPlanningResultToConversation method
func (m *MockAIPlanningEngine) LinkPlanningResultToConversation(ctx context.Context, planningResultID, conversationID string) error {
	args := m.Called(ctx, planningResultID, conversationID)
	return args.Error(0)
}

// MockExecutionCoordinator provides a testify-based mock for execution coordination
type MockExecutionCoordinator struct {
	mock.Mock
}

// NewMockExecutionCoordinator creates a new mock execution coordinator instance
func NewMockExecutionCoordinator() *MockExecutionCoordinator {
	return &MockExecutionCoordinator{}
}

// StartExecution mocks the StartExecution method for async execution
func (m *MockExecutionCoordinator) StartExecution(ctx context.Context, planID string) error {
	args := m.Called(ctx, planID)
	return args.Error(0)
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
