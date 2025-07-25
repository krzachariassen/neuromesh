package domain

import (
	"context"

	executionDomain "neuromesh/internal/execution/domain"
)

// ExecutionPlanRepository defines the interface for execution plan persistence
type ExecutionPlanRepository interface {
	// Plan operations
	Create(ctx context.Context, plan *ExecutionPlan) error
	GetByID(ctx context.Context, id string) (*ExecutionPlan, error)
	GetByAnalysisID(ctx context.Context, analysisID string) (*ExecutionPlan, error)
	Update(ctx context.Context, plan *ExecutionPlan) error

	// Relationship operations
	LinkToAnalysis(ctx context.Context, analysisID, planID string) error

	// Step operations
	GetStepsByPlanID(ctx context.Context, planID string) ([]*ExecutionStep, error)
	AddStep(ctx context.Context, step *ExecutionStep) error
	UpdateStep(ctx context.Context, step *ExecutionStep) error
	AssignStepToAgent(ctx context.Context, stepID, agentID string) error

	// Agent Result operations - NEW for graph-native result synthesis
	StoreAgentResult(ctx context.Context, result *executionDomain.AgentResult) error
	GetAgentResultsByExecutionPlan(ctx context.Context, planID string) ([]*executionDomain.AgentResult, error)
	GetAgentResultsByExecutionStep(ctx context.Context, stepID string) ([]*executionDomain.AgentResult, error)
	GetAgentResultByID(ctx context.Context, resultID string) (*executionDomain.AgentResult, error)
}
