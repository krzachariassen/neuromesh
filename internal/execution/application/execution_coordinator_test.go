package application

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"neuromesh/internal/execution/domain"
	planningDomain "neuromesh/internal/planning/domain"
)

// MockExecutionPlanRepository provides a testify-based mock for execution plan repository operations
type MockExecutionPlanRepository struct {
	mock.Mock
}

func (m *MockExecutionPlanRepository) Create(ctx context.Context, plan *planningDomain.ExecutionPlan) error {
	args := m.Called(ctx, plan)
	return args.Error(0)
}

func (m *MockExecutionPlanRepository) GetByID(ctx context.Context, id string) (*planningDomain.ExecutionPlan, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*planningDomain.ExecutionPlan), args.Error(1)
}

func (m *MockExecutionPlanRepository) GetByAnalysisID(ctx context.Context, analysisID string) (*planningDomain.ExecutionPlan, error) {
	args := m.Called(ctx, analysisID)
	return args.Get(0).(*planningDomain.ExecutionPlan), args.Error(1)
}

func (m *MockExecutionPlanRepository) Update(ctx context.Context, plan *planningDomain.ExecutionPlan) error {
	args := m.Called(ctx, plan)
	return args.Error(0)
}

func (m *MockExecutionPlanRepository) LinkToAnalysis(ctx context.Context, analysisID, planID string) error {
	args := m.Called(ctx, analysisID, planID)
	return args.Error(0)
}

func (m *MockExecutionPlanRepository) GetStepsByPlanID(ctx context.Context, planID string) ([]*planningDomain.ExecutionStep, error) {
	args := m.Called(ctx, planID)
	return args.Get(0).([]*planningDomain.ExecutionStep), args.Error(1)
}

func (m *MockExecutionPlanRepository) AddStep(ctx context.Context, step *planningDomain.ExecutionStep) error {
	args := m.Called(ctx, step)
	return args.Error(0)
}

func (m *MockExecutionPlanRepository) UpdateStep(ctx context.Context, step *planningDomain.ExecutionStep) error {
	args := m.Called(ctx, step)
	return args.Error(0)
}

func (m *MockExecutionPlanRepository) AssignStepToAgent(ctx context.Context, stepID, agentID string) error {
	args := m.Called(ctx, stepID, agentID)
	return args.Error(0)
}

func (m *MockExecutionPlanRepository) StoreAgentResult(ctx context.Context, result *domain.AgentResult) error {
	args := m.Called(ctx, result)
	return args.Error(0)
}

func (m *MockExecutionPlanRepository) GetAgentResultsByExecutionPlan(ctx context.Context, planID string) ([]*domain.AgentResult, error) {
	args := m.Called(ctx, planID)
	return args.Get(0).([]*domain.AgentResult), args.Error(1)
}

func (m *MockExecutionPlanRepository) GetAgentResultsByExecutionStep(ctx context.Context, stepID string) ([]*domain.AgentResult, error) {
	args := m.Called(ctx, stepID)
	return args.Get(0).([]*domain.AgentResult), args.Error(1)
}

func (m *MockExecutionPlanRepository) GetAgentResultByID(ctx context.Context, resultID string) (*domain.AgentResult, error) {
	args := m.Called(ctx, resultID)
	return args.Get(0).(*domain.AgentResult), args.Error(1)
}

// MockResultSynthesizer provides a testify-based mock for result synthesizer operations
type MockResultSynthesizer struct {
	mock.Mock
}

func (m *MockResultSynthesizer) SynthesizeResults(ctx context.Context, planID string) (string, error) {
	args := m.Called(ctx, planID)
	return args.String(0), args.Error(1)
}

func (m *MockResultSynthesizer) GetSynthesisContext(ctx context.Context, planID string) (*domain.SynthesisContext, error) {
	args := m.Called(ctx, planID)
	return args.Get(0).(*domain.SynthesisContext), args.Error(1)
}

func TestExecutionCoordinator_IsExecutionPlanComplete(t *testing.T) {
	tests := []struct {
		name           string
		setupMocks     func(*MockExecutionPlanRepository)
		planID         string
		expectedResult bool
		expectedError  string
	}{
		{
			name: "should return true when all steps have successful results",
			setupMocks: func(repo *MockExecutionPlanRepository) {
				steps := []*planningDomain.ExecutionStep{
					{ID: "step1", Status: planningDomain.ExecutionStepStatusCompleted},
					{ID: "step2", Status: planningDomain.ExecutionStepStatusCompleted},
				}
				repo.On("GetStepsByPlanID", mock.Anything, "plan-1").Return(steps, nil)

				results1 := []*domain.AgentResult{
					{ID: "result1", ExecutionStepID: "step1", Status: domain.AgentResultStatusSuccess},
				}
				results2 := []*domain.AgentResult{
					{ID: "result2", ExecutionStepID: "step2", Status: domain.AgentResultStatusSuccess},
				}
				repo.On("GetAgentResultsByExecutionStep", mock.Anything, "step1").Return(results1, nil)
				repo.On("GetAgentResultsByExecutionStep", mock.Anything, "step2").Return(results2, nil)
			},
			planID:         "plan-1",
			expectedResult: true,
			expectedError:  "",
		},
		{
			name: "should return false when some steps are still pending",
			setupMocks: func(repo *MockExecutionPlanRepository) {
				steps := []*planningDomain.ExecutionStep{
					{ID: "step1", Status: planningDomain.ExecutionStepStatusCompleted},
					{ID: "step2", Status: planningDomain.ExecutionStepStatusPending},
				}
				repo.On("GetStepsByPlanID", mock.Anything, "plan-1").Return(steps, nil)

				results1 := []*domain.AgentResult{
					{ID: "result1", ExecutionStepID: "step1", Status: domain.AgentResultStatusSuccess},
				}
				repo.On("GetAgentResultsByExecutionStep", mock.Anything, "step1").Return(results1, nil)
			},
			planID:         "plan-1",
			expectedResult: false,
			expectedError:  "",
		},
		{
			name: "should return false when some steps have failed results",
			setupMocks: func(repo *MockExecutionPlanRepository) {
				steps := []*planningDomain.ExecutionStep{
					{ID: "step1", Status: planningDomain.ExecutionStepStatusCompleted},
					{ID: "step2", Status: planningDomain.ExecutionStepStatusCompleted},
				}
				repo.On("GetStepsByPlanID", mock.Anything, "plan-1").Return(steps, nil)

				results1 := []*domain.AgentResult{
					{ID: "result1", ExecutionStepID: "step1", Status: domain.AgentResultStatusSuccess},
				}
				results2 := []*domain.AgentResult{
					{ID: "result2", ExecutionStepID: "step2", Status: domain.AgentResultStatusFailed},
				}
				repo.On("GetAgentResultsByExecutionStep", mock.Anything, "step1").Return(results1, nil)
				repo.On("GetAgentResultsByExecutionStep", mock.Anything, "step2").Return(results2, nil)
			},
			planID:         "plan-1",
			expectedResult: false,
			expectedError:  "",
		},
		{
			name: "should return error when repository fails",
			setupMocks: func(repo *MockExecutionPlanRepository) {
				repo.On("GetStepsByPlanID", mock.Anything, "plan-1").Return([]*planningDomain.ExecutionStep{}, errors.New("database error"))
			},
			planID:         "plan-1",
			expectedResult: false,
			expectedError:  "database error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockRepo := &MockExecutionPlanRepository{}
			mockSynthesizer := &MockResultSynthesizer{}
			tt.setupMocks(mockRepo)

			coordinator := NewExecutionCoordinator(mockRepo, mockSynthesizer)
			ctx := context.Background()

			// Act
			result, err := coordinator.IsExecutionPlanComplete(ctx, tt.planID)

			// Assert
			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResult, result)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestExecutionCoordinator_TriggerSynthesisWhenComplete(t *testing.T) {
	tests := []struct {
		name           string
		setupMocks     func(*MockExecutionPlanRepository, *MockResultSynthesizer)
		planID         string
		expectedResult string
		expectedError  string
	}{
		{
			name: "should trigger synthesis when execution plan is complete",
			setupMocks: func(repo *MockExecutionPlanRepository, synthesizer *MockResultSynthesizer) {
				steps := []*planningDomain.ExecutionStep{
					{ID: "step1", Status: planningDomain.ExecutionStepStatusCompleted},
				}
				repo.On("GetStepsByPlanID", mock.Anything, "plan-1").Return(steps, nil)

				results := []*domain.AgentResult{
					{ID: "result1", ExecutionStepID: "step1", Status: domain.AgentResultStatusSuccess},
				}
				repo.On("GetAgentResultsByExecutionStep", mock.Anything, "step1").Return(results, nil)

				synthesizer.On("SynthesizeResults", mock.Anything, "plan-1").Return("Synthesized diagnostic report", nil)
			},
			planID:         "plan-1",
			expectedResult: "Synthesized diagnostic report",
			expectedError:  "",
		},
		{
			name: "should return empty when execution plan is not complete",
			setupMocks: func(repo *MockExecutionPlanRepository, synthesizer *MockResultSynthesizer) {
				steps := []*planningDomain.ExecutionStep{
					{ID: "step1", Status: planningDomain.ExecutionStepStatusPending},
				}
				repo.On("GetStepsByPlanID", mock.Anything, "plan-1").Return(steps, nil)
			},
			planID:         "plan-1",
			expectedResult: "",
			expectedError:  "",
		},
		{
			name: "should return error when synthesis fails",
			setupMocks: func(repo *MockExecutionPlanRepository, synthesizer *MockResultSynthesizer) {
				steps := []*planningDomain.ExecutionStep{
					{ID: "step1", Status: planningDomain.ExecutionStepStatusCompleted},
				}
				repo.On("GetStepsByPlanID", mock.Anything, "plan-1").Return(steps, nil)

				results := []*domain.AgentResult{
					{ID: "result1", ExecutionStepID: "step1", Status: domain.AgentResultStatusSuccess},
				}
				repo.On("GetAgentResultsByExecutionStep", mock.Anything, "step1").Return(results, nil)

				synthesizer.On("SynthesizeResults", mock.Anything, "plan-1").Return("", errors.New("AI synthesis failed"))
			},
			planID:         "plan-1",
			expectedResult: "",
			expectedError:  "AI synthesis failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockRepo := &MockExecutionPlanRepository{}
			mockSynthesizer := &MockResultSynthesizer{}
			tt.setupMocks(mockRepo, mockSynthesizer)

			coordinator := NewExecutionCoordinator(mockRepo, mockSynthesizer)
			ctx := context.Background()

			// Act
			result, err := coordinator.TriggerSynthesisWhenComplete(ctx, tt.planID)

			// Assert
			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResult, result)
			}

			mockRepo.AssertExpectations(t)
			mockSynthesizer.AssertExpectations(t)
		})
	}
}

func TestExecutionCoordinator_HandlePartialCompletion(t *testing.T) {
	tests := []struct {
		name          string
		setupMocks    func(*MockExecutionPlanRepository)
		planID        string
		expectedStats *domain.ExecutionStats
		expectedError string
	}{
		{
			name: "should return correct stats for mixed execution results",
			setupMocks: func(repo *MockExecutionPlanRepository) {
				steps := []*planningDomain.ExecutionStep{
					{ID: "step1", Status: planningDomain.ExecutionStepStatusCompleted},
					{ID: "step2", Status: planningDomain.ExecutionStepStatusCompleted},
					{ID: "step3", Status: planningDomain.ExecutionStepStatusPending},
				}
				repo.On("GetStepsByPlanID", mock.Anything, "plan-1").Return(steps, nil)

				results1 := []*domain.AgentResult{
					{ID: "result1", ExecutionStepID: "step1", Status: domain.AgentResultStatusSuccess},
				}
				results2 := []*domain.AgentResult{
					{ID: "result2", ExecutionStepID: "step2", Status: domain.AgentResultStatusFailed},
				}
				repo.On("GetAgentResultsByExecutionStep", mock.Anything, "step1").Return(results1, nil)
				repo.On("GetAgentResultsByExecutionStep", mock.Anything, "step2").Return(results2, nil)
			},
			planID: "plan-1",
			expectedStats: &domain.ExecutionStats{
				TotalSteps:        3,
				CompletedSteps:    2,
				PendingSteps:      1,
				SuccessfulResults: 1,
				FailedResults:     1,
				PartialResults:    0,
			},
			expectedError: "",
		},
		{
			name: "should handle empty execution plan",
			setupMocks: func(repo *MockExecutionPlanRepository) {
				repo.On("GetStepsByPlanID", mock.Anything, "plan-1").Return([]*planningDomain.ExecutionStep{}, nil)
			},
			planID: "plan-1",
			expectedStats: &domain.ExecutionStats{
				TotalSteps:        0,
				CompletedSteps:    0,
				PendingSteps:      0,
				SuccessfulResults: 0,
				FailedResults:     0,
				PartialResults:    0,
			},
			expectedError: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockRepo := &MockExecutionPlanRepository{}
			mockSynthesizer := &MockResultSynthesizer{}
			tt.setupMocks(mockRepo)

			coordinator := NewExecutionCoordinator(mockRepo, mockSynthesizer)
			ctx := context.Background()

			// Act
			stats, err := coordinator.HandlePartialCompletion(ctx, tt.planID)

			// Assert
			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedStats, stats)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}
