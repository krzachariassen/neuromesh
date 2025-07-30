package domain

import (
	"context"

	executionDomain "neuromesh/internal/execution/domain"
)

// ExecutionPlanRepository defines the interface for unified execution plan persistence
// This consolidates planning and execution operations into a single repository
type ExecutionPlanRepository interface {
	// Unified Plan operations (consolidates planning result + execution plan)
	Create(ctx context.Context, plan *ExecutionPlan) error
	GetByID(ctx context.Context, id string) (*ExecutionPlan, error)
	GetByRequestID(ctx context.Context, requestID string) ([]*ExecutionPlan, error) // From PlanningResultRepository
	GetByAnalysisID(ctx context.Context, analysisID string) (*ExecutionPlan, error)
	Update(ctx context.Context, plan *ExecutionPlan) error
	Delete(ctx context.Context, id string) error // From PlanningResultRepository

	// Relationship operations (consolidated)
	LinkToAnalysis(ctx context.Context, analysisID, planID string) error
	LinkToRequest(ctx context.Context, planID, requestID string) error           // From PlanningResultRepository
	LinkToConversation(ctx context.Context, planID, conversationID string) error // From PlanningResultRepository

	// Step operations
	GetStepsByPlanID(ctx context.Context, planID string) ([]*ExecutionStep, error)
	AddStep(ctx context.Context, step *ExecutionStep) error
	UpdateStep(ctx context.Context, step *ExecutionStep) error
	AssignStepToAgent(ctx context.Context, stepID, agentID string) error

	// Agent Result operations - graph-native result synthesis
	StoreAgentResult(ctx context.Context, result *executionDomain.AgentResult) error
	GetAgentResultsByExecutionPlan(ctx context.Context, planID string) ([]*executionDomain.AgentResult, error)
	GetAgentResultsByExecutionStep(ctx context.Context, stepID string) ([]*executionDomain.AgentResult, error)
	GetAgentResultByID(ctx context.Context, resultID string) (*executionDomain.AgentResult, error)

	// Correlation mapping operations - agent result linking
	GetPlanIDByCorrelationID(ctx context.Context, correlationID string) (string, error)

	// Synthesis Result operations - synthesis result storage
	StoreSynthesisResult(ctx context.Context, result *executionDomain.SynthesisResult) error
	GetSynthesisResultByPlanID(ctx context.Context, planID string) (*executionDomain.SynthesisResult, error)
}
