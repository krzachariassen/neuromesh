package application

import (
	"context"
	"testing"

	executionDomain "neuromesh/internal/execution/domain"
	planningDomain "neuromesh/internal/planning/domain"
)

// RED: Write failing test that exposes the design need for synthesis result storage
func TestSynthesisResultStorage_TDD(t *testing.T) {
	// Setup test infrastructure
	repository := &TestSynthesisRepository{
		plans:            make(map[string]*planningDomain.ExecutionPlan),
		planSteps:        make(map[string][]*planningDomain.ExecutionStep),
		agentResults:     []*executionDomain.AgentResult{},
		synthesisResults: []*executionDomain.SynthesisResult{},
	}

	synthesizer := &TestSynthesizer{}

	coordinator := &ExecutionCoordinator{
		executionRepo: repository,
		synthesizer:   synthesizer,
	}

	// Test scenario: Store synthesis result after agent results are available
	ctx := context.Background()
	planID := "test-plan-123"

	// Setup: Create a plan with some agent results
	plan := &planningDomain.ExecutionPlan{
		ID:     planID,
		Name:   "Test Plan",
		Status: planningDomain.ExecutionPlanStatusExecuting,
	}
	repository.plans[planID] = plan

	// Setup: Add some agent results to synthesize
	agentResult1 := &executionDomain.AgentResult{
		ID:              "result-1",
		ExecutionStepID: "step-1",
		Content:         "Text analysis: The phrase contains 7 words.",
		Status:          executionDomain.AgentResultStatusSuccess,
	}
	agentResult2 := &executionDomain.AgentResult{
		ID:              "result-2",
		ExecutionStepID: "step-2",
		Content:         "Grammar check: All words are properly formatted.",
		Status:          executionDomain.AgentResultStatusSuccess,
	}
	repository.agentResults = append(repository.agentResults, agentResult1, agentResult2)

	// GREEN PHASE: This should now pass because StoreSynthesisResult is implemented
	err := coordinator.StoreSynthesisResult(ctx, planID)
	if err != nil {
		t.Errorf("Expected no error with implemented StoreSynthesisResult, but got: %v", err)
	}

	// GREEN VALIDATION: Verify synthesis result was stored
	synthesisResult, err := repository.GetSynthesisResultByPlanID(ctx, planID)
	if err != nil {
		t.Errorf("Repository should return synthesis result without error, got error: %v", err)
	}
	if synthesisResult == nil {
		t.Errorf("Expected synthesis result to be stored, but found none")
	}
	if synthesisResult != nil && synthesisResult.PlanID != planID {
		t.Errorf("Expected synthesis result plan ID %s, got %s", planID, synthesisResult.PlanID)
	}
	if synthesisResult != nil && synthesisResult.Content == "" {
		t.Errorf("Expected synthesis result to have content, but got empty string")
	}
}

// TestSynthesisRepository provides a test implementation of ExecutionPlanRepository
type TestSynthesisRepository struct {
	plans            map[string]*planningDomain.ExecutionPlan
	planSteps        map[string][]*planningDomain.ExecutionStep
	agentResults     []*executionDomain.AgentResult
	synthesisResults []*executionDomain.SynthesisResult
}

func (m *TestSynthesisRepository) StoreSynthesisResult(ctx context.Context, result *executionDomain.SynthesisResult) error {
	m.synthesisResults = append(m.synthesisResults, result)
	return nil
}

func (m *TestSynthesisRepository) GetSynthesisResultByPlanID(ctx context.Context, planID string) (*executionDomain.SynthesisResult, error) {
	for _, result := range m.synthesisResults {
		if result.PlanID == planID {
			return result, nil
		}
	}
	return nil, nil
}

func (m *TestSynthesisRepository) GetStepsByPlanID(ctx context.Context, planID string) ([]*planningDomain.ExecutionStep, error) {
	return m.planSteps[planID], nil
}

func (m *TestSynthesisRepository) GetAgentResultsByExecutionPlan(ctx context.Context, planID string) ([]*executionDomain.AgentResult, error) {
	return m.agentResults, nil
}

func (m *TestSynthesisRepository) GetAgentResultByID(ctx context.Context, resultID string) (*executionDomain.AgentResult, error) {
	for _, result := range m.agentResults {
		if result.ID == resultID {
			return result, nil
		}
	}
	return nil, nil
}

func (m *TestSynthesisRepository) GetAgentResultsByExecutionStep(ctx context.Context, stepID string) ([]*executionDomain.AgentResult, error) {
	var results []*executionDomain.AgentResult
	for _, result := range m.agentResults {
		if result.ExecutionStepID == stepID {
			results = append(results, result)
		}
	}
	return results, nil
}

func (m *TestSynthesisRepository) GetPlanIDByCorrelationID(ctx context.Context, correlationID string) (string, error) {
	return "", nil
}

// Implement other required methods as no-ops for now
func (m *TestSynthesisRepository) StoreAgentResult(ctx context.Context, result *executionDomain.AgentResult) error {
	return nil
}
func (m *TestSynthesisRepository) UpdateStep(ctx context.Context, step *planningDomain.ExecutionStep) error {
	return nil
}
func (m *TestSynthesisRepository) Create(ctx context.Context, plan *planningDomain.ExecutionPlan) error {
	return nil
}
func (m *TestSynthesisRepository) GetByID(ctx context.Context, id string) (*planningDomain.ExecutionPlan, error) {
	return m.plans[id], nil
}
func (m *TestSynthesisRepository) GetByAnalysisID(ctx context.Context, analysisID string) (*planningDomain.ExecutionPlan, error) {
	return nil, nil
}
func (m *TestSynthesisRepository) Update(ctx context.Context, plan *planningDomain.ExecutionPlan) error {
	return nil
}
func (m *TestSynthesisRepository) LinkToAnalysis(ctx context.Context, analysisID, planID string) error {
	return nil
}
func (m *TestSynthesisRepository) AddStep(ctx context.Context, step *planningDomain.ExecutionStep) error {
	return nil
}
func (m *TestSynthesisRepository) AssignStepToAgent(ctx context.Context, stepID, agentID string) error {
	return nil
}

// TestSynthesizer provides a test implementation of synthesis
type TestSynthesizer struct{}

func (s *TestSynthesizer) SynthesizeResults(ctx context.Context, planID string) (string, error) {
	return "Synthesized result: The text contains 7 words based on agent analysis.", nil
}

func (s *TestSynthesizer) GetSynthesisContext(ctx context.Context, planID string) (*executionDomain.SynthesisContext, error) {
	return &executionDomain.SynthesisContext{
		ExecutionPlanID: planID,
		AgentResults: []*executionDomain.AgentResult{
			{
				ID:              "result-1",
				ExecutionStepID: "step-1",
				Content:         "Text analysis: The phrase contains 7 words.",
				Status:          executionDomain.AgentResultStatusSuccess,
			},
		},
	}, nil
}
