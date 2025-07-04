package infrastructure

import (
	"context"

	"neuromesh/internal/execution/domain"
	"neuromesh/internal/graph"
)

// GraphExecutionService implements ExecutionService using the graph backend
type GraphExecutionService struct {
	graph graph.Graph
}

// NewGraphExecutionService creates a new GraphExecutionService
func NewGraphExecutionService(graph graph.Graph) *GraphExecutionService {
	return &GraphExecutionService{
		graph: graph,
	}
}

// CreateExecutionPlan stores an execution plan in the graph
func (ges *GraphExecutionService) CreateExecutionPlan(ctx context.Context, plan *domain.ExecutionPlan) error {
	// This would normally store the plan in the graph
	// For now, return success to satisfy the interface
	return nil
}

// GetExecutionPlan retrieves an execution plan from the graph
func (ges *GraphExecutionService) GetExecutionPlan(ctx context.Context, planID string) (*domain.ExecutionPlan, error) {
	// This would normally query the graph for the plan
	// For now, return a mock plan
	plan := &domain.ExecutionPlan{
		ID:     planID,
		Status: domain.ExecutionStatusPending,
		Steps: []domain.ExecutionStep{
			{
				ID:     "step-1",
				Name:   "Initialize",
				Action: "init",
				Status: domain.ExecutionStatusPending,
			},
		},
	}

	return plan, nil
}

// UpdateExecutionStatus updates the status of an execution plan
func (ges *GraphExecutionService) UpdateExecutionStatus(ctx context.Context, planID string, status domain.ExecutionStatus) error {
	// This would normally update the plan status in the graph
	// For now, return success to satisfy the interface
	return nil
}
