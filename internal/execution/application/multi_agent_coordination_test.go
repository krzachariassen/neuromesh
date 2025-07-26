package application

import (
	"context"
	"testing"

	"neuromesh/internal/execution/domain"
	"neuromesh/internal/messaging"
	planningDomain "neuromesh/internal/planning/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// TestMultiAgentCoordination demonstrates how multiple agents coordinate to trigger synthesis
func TestMultiAgentCoordination(t *testing.T) {
	t.Run("multiple agents coordinate to trigger synthesis when all complete", func(t *testing.T) {
		// Create mocks
		mockRepository := &mockRepository{}
		mockSynthesizer := &mockSynthesizer{}
		mockMessageBus := &mockMessageBus{}

		// Create real coordinator with mock dependencies
		coordinator := NewExecutionCoordinator(mockRepository, mockSynthesizer)

		// Create synthesis event handler
		handler := NewSynthesisEventHandler(coordinator, mockMessageBus, mockRepository, mockSynthesizer)

		// Simulate a plan with 3 steps (3 agents)
		planID := "healthcare-plan-123"

		// Set up mock expectations for the first agent completion
		// For agent 1 completion check - return 2 incomplete steps
		mockRepository.On("GetStepsByPlanID", mock.Anything, planID).Return([]*planningDomain.ExecutionStep{
			{ID: "step-1", Status: planningDomain.ExecutionStepStatusCompleted}, // Agent 1 done
			{ID: "step-2", Status: planningDomain.ExecutionStepStatusExecuting}, // Agent 2 still working
			{ID: "step-3", Status: planningDomain.ExecutionStepStatusAssigned},  // Agent 3 not started
		}, nil).Once()

		// Mock agent result for step-1 (needed for IsExecutionPlanComplete check)
		mockRepository.On("GetAgentResultsByExecutionStep", mock.Anything, "step-1").Return([]*domain.AgentResult{
			{Status: domain.AgentResultStatusSuccess},
		}, nil).Once()

		event1 := &AgentCompletedEvent{
			PlanID:  planID,
			StepID:  "step-1",
			AgentID: "agent-symptom-analyzer",
		}

		err := handler.HandleAgentCompleted(context.Background(), event1)
		assert.NoError(t, err)

		// Set up mock expectations for the second agent completion
		// For agent 2 completion check - return 1 incomplete step
		mockRepository.On("GetStepsByPlanID", mock.Anything, planID).Return([]*planningDomain.ExecutionStep{
			{ID: "step-1", Status: planningDomain.ExecutionStepStatusCompleted}, // Agent 1 done
			{ID: "step-2", Status: planningDomain.ExecutionStepStatusCompleted}, // Agent 2 done
			{ID: "step-3", Status: planningDomain.ExecutionStepStatusExecuting}, // Agent 3 still working
		}, nil).Once()

		// Mock agent results for completed steps (needed for IsExecutionPlanComplete check)
		mockRepository.On("GetAgentResultsByExecutionStep", mock.Anything, "step-1").Return([]*domain.AgentResult{
			{Status: domain.AgentResultStatusSuccess},
		}, nil).Once()
		mockRepository.On("GetAgentResultsByExecutionStep", mock.Anything, "step-2").Return([]*domain.AgentResult{
			{Status: domain.AgentResultStatusSuccess},
		}, nil).Once()

		event2 := &AgentCompletedEvent{
			PlanID:  planID,
			StepID:  "step-2",
			AgentID: "agent-diagnostic-specialist",
		}

		err = handler.HandleAgentCompleted(context.Background(), event2)
		assert.NoError(t, err)

		// Set up mock expectations for the third agent completion (this triggers synthesis)
		// For agent 3 completion check - return ALL steps completed
		// NOTE: GetStepsByPlanID will be called TWICE for this event:
		// 1. First by HandleAgentCompleted -> IsExecutionPlanComplete
		// 2. Then by TriggerSynthesisWhenComplete -> IsExecutionPlanComplete
		mockRepository.On("GetStepsByPlanID", mock.Anything, planID).Return([]*planningDomain.ExecutionStep{
			{ID: "step-1", Status: planningDomain.ExecutionStepStatusCompleted}, // Agent 1 done
			{ID: "step-2", Status: planningDomain.ExecutionStepStatusCompleted}, // Agent 2 done
			{ID: "step-3", Status: planningDomain.ExecutionStepStatusCompleted}, // Agent 3 done
		}, nil).Twice() // Called twice: once for check, once for synthesis trigger

		// Mock agent results for each step (all successful) - this will trigger synthesis
		// NOTE: These will also be called TWICE - once for each IsExecutionPlanComplete call
		mockRepository.On("GetAgentResultsByExecutionStep", mock.Anything, "step-1").Return([]*domain.AgentResult{
			{Status: domain.AgentResultStatusSuccess},
		}, nil).Twice()
		mockRepository.On("GetAgentResultsByExecutionStep", mock.Anything, "step-2").Return([]*domain.AgentResult{
			{Status: domain.AgentResultStatusSuccess},
		}, nil).Twice()
		mockRepository.On("GetAgentResultsByExecutionStep", mock.Anything, "step-3").Return([]*domain.AgentResult{
			{Status: domain.AgentResultStatusSuccess},
		}, nil).Twice()

		// Mock the synthesis call
		mockSynthesizer.On("SynthesizeResults", mock.Anything, planID).Return("Complete healthcare diagnosis synthesized from all agents", nil).Once()

		event3 := &AgentCompletedEvent{
			PlanID:  planID,
			StepID:  "step-3",
			AgentID: "agent-treatment-advisor",
		}

		err = handler.HandleAgentCompleted(context.Background(), event3)
		assert.NoError(t, err)

		// Verify that synthesis WAS triggered when all agents completed
		mockRepository.AssertExpectations(t)
		mockSynthesizer.AssertExpectations(t)
	})
}

// Mock implementations for the test (simplified versions)
type mockCoordinator struct {
	mock.Mock
}

func (m *mockCoordinator) IsExecutionPlanComplete(ctx context.Context, planID string) (bool, error) {
	args := m.Called(ctx, planID)
	return args.Bool(0), args.Error(1)
}

func (m *mockCoordinator) TriggerSynthesisWhenComplete(ctx context.Context, planID string) (string, error) {
	args := m.Called(ctx, planID)
	return args.String(0), args.Error(1)
}

func (m *mockCoordinator) HandlePartialCompletion(ctx context.Context, planID string) (*domain.ExecutionStats, error) {
	args := m.Called(ctx, planID)
	return args.Get(0).(*domain.ExecutionStats), args.Error(1)
}

type mockMessageBus struct {
	mock.Mock
}

func (m *mockMessageBus) SendToAgent(ctx context.Context, msg *messaging.AIToAgentMessage) error {
	args := m.Called(ctx, msg)
	return args.Error(0)
}

func (m *mockMessageBus) SendToAI(ctx context.Context, msg *messaging.AgentToAIMessage) error {
	args := m.Called(ctx, msg)
	return args.Error(0)
}

func (m *mockMessageBus) SendBetweenAgents(ctx context.Context, msg *messaging.AgentToAgentMessage) error {
	args := m.Called(ctx, msg)
	return args.Error(0)
}

func (m *mockMessageBus) SendUserToAI(ctx context.Context, msg *messaging.UserToAIMessage) error {
	args := m.Called(ctx, msg)
	return args.Error(0)
}

func (m *mockMessageBus) Subscribe(ctx context.Context, participantID string) (<-chan *messaging.Message, error) {
	args := m.Called(ctx, participantID)
	return args.Get(0).(<-chan *messaging.Message), args.Error(1)
}

func (m *mockMessageBus) GetConversationHistory(ctx context.Context, correlationID string) ([]*messaging.Message, error) {
	args := m.Called(ctx, correlationID)
	return args.Get(0).([]*messaging.Message), args.Error(1)
}

func (m *mockMessageBus) PrepareAgentQueue(ctx context.Context, agentID string) error {
	args := m.Called(ctx, agentID)
	return args.Error(0)
}

type mockRepository struct {
	mock.Mock
}

func (m *mockRepository) Store(ctx context.Context, plan *planningDomain.ExecutionPlan) error {
	args := m.Called(ctx, plan)
	return args.Error(0)
}

func (m *mockRepository) GetByID(ctx context.Context, id string) (*planningDomain.ExecutionPlan, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*planningDomain.ExecutionPlan), args.Error(1)
}

func (m *mockRepository) Update(ctx context.Context, plan *planningDomain.ExecutionPlan) error {
	args := m.Called(ctx, plan)
	return args.Error(0)
}

func (m *mockRepository) StoreAgentResult(ctx context.Context, result *domain.AgentResult) error {
	args := m.Called(ctx, result)
	return args.Error(0)
}

func (m *mockRepository) GetAgentResultsByExecutionPlan(ctx context.Context, planID string) ([]*domain.AgentResult, error) {
	args := m.Called(ctx, planID)
	return args.Get(0).([]*domain.AgentResult), args.Error(1)
}

func (m *mockRepository) GetAgentResultsByExecutionStep(ctx context.Context, stepID string) ([]*domain.AgentResult, error) {
	args := m.Called(ctx, stepID)
	return args.Get(0).([]*domain.AgentResult), args.Error(1)
}

func (m *mockRepository) GetStepsByPlanID(ctx context.Context, planID string) ([]*planningDomain.ExecutionStep, error) {
	args := m.Called(ctx, planID)
	return args.Get(0).([]*planningDomain.ExecutionStep), args.Error(1)
}

// Additional required methods for the interface
func (m *mockRepository) Create(ctx context.Context, plan *planningDomain.ExecutionPlan) error {
	args := m.Called(ctx, plan)
	return args.Error(0)
}

func (m *mockRepository) GetByAnalysisID(ctx context.Context, analysisID string) (*planningDomain.ExecutionPlan, error) {
	args := m.Called(ctx, analysisID)
	return args.Get(0).(*planningDomain.ExecutionPlan), args.Error(1)
}

func (m *mockRepository) LinkToAnalysis(ctx context.Context, analysisID, planID string) error {
	args := m.Called(ctx, analysisID, planID)
	return args.Error(0)
}

func (m *mockRepository) AddStep(ctx context.Context, step *planningDomain.ExecutionStep) error {
	args := m.Called(ctx, step)
	return args.Error(0)
}

func (m *mockRepository) UpdateStep(ctx context.Context, step *planningDomain.ExecutionStep) error {
	args := m.Called(ctx, step)
	return args.Error(0)
}

func (m *mockRepository) AssignStepToAgent(ctx context.Context, stepID, agentID string) error {
	args := m.Called(ctx, stepID, agentID)
	return args.Error(0)
}

func (m *mockRepository) GetAgentResultByID(ctx context.Context, resultID string) (*domain.AgentResult, error) {
	args := m.Called(ctx, resultID)
	return args.Get(0).(*domain.AgentResult), args.Error(1)
}

type mockSynthesizer struct {
	mock.Mock
}

func (m *mockSynthesizer) SynthesizeResults(ctx context.Context, planID string) (string, error) {
	args := m.Called(ctx, planID)
	return args.String(0), args.Error(1)
}

func (m *mockSynthesizer) GetSynthesisContext(ctx context.Context, planID string) (*domain.SynthesisContext, error) {
	args := m.Called(ctx, planID)
	return args.Get(0).(*domain.SynthesisContext), args.Error(1)
}
