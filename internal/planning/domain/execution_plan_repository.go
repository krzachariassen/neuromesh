package domain

import "context"

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
}
