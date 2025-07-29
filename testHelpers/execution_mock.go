package testHelpers

import (
	"context"

	"neuromesh/internal/execution/domain"

	"github.com/stretchr/testify/mock"
)

// MockResultSynthesizer provides a testify-based mock for result synthesis operations
type MockResultSynthesizer struct {
	mock.Mock
}

// NewMockResultSynthesizer creates a new mock result synthesizer instance
func NewMockResultSynthesizer() *MockResultSynthesizer {
	return &MockResultSynthesizer{}
}

func (m *MockResultSynthesizer) SynthesizeResults(ctx context.Context, planID string) (string, error) {
	args := m.Called(ctx, planID)
	return args.String(0), args.Error(1)
}

func (m *MockResultSynthesizer) GetSynthesisContext(ctx context.Context, planID string) (*domain.SynthesisContext, error) {
	args := m.Called(ctx, planID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.SynthesisContext), args.Error(1)
}

// MockAIExecutionEngine provides a testify-based mock for AI execution engine operations
type MockAIExecutionEngine struct {
	mock.Mock
}

// NewMockAIExecutionEngine creates a new mock AI execution engine instance
func NewMockAIExecutionEngine() *MockAIExecutionEngine {
	return &MockAIExecutionEngine{}
}

func (m *MockAIExecutionEngine) ExecutePlan(ctx context.Context, planID string) error {
	args := m.Called(ctx, planID)
	return args.Error(0)
}

func (m *MockAIExecutionEngine) ExecuteWithAgents(ctx context.Context, executionPlan, userInput, userID, agentContext string) (string, error) {
	args := m.Called(ctx, executionPlan, userInput, userID, agentContext)
	return args.String(0), args.Error(1)
}

// SetExecuteWithAgentsResult sets up mock result for ExecuteWithAgents
func (m *MockAIExecutionEngine) SetExecuteWithAgentsResult(result string, err error) {
	m.On("ExecuteWithAgents", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(result, err)
}
