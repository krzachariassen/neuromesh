package domain

import "context"

// PlanningResultRepository defines the interface for persisting planning results
// This replaces decision repository with proper planning terminology
type PlanningResultRepository interface {
	// Store persists a planning result
	Store(ctx context.Context, result *PlanningResult) error

	// GetByID retrieves a planning result by ID
	GetByID(ctx context.Context, id string) (*PlanningResult, error)

	// GetByRequestID retrieves planning results for a specific request
	GetByRequestID(ctx context.Context, requestID string) ([]*PlanningResult, error)

	// Update updates an existing planning result
	Update(ctx context.Context, result *PlanningResult) error

	// Delete removes a planning result
	Delete(ctx context.Context, id string) error

	// LinkToRequest links a planning result to a request
	LinkToRequest(ctx context.Context, planningResultID, requestID string) error

	// LinkToExecutionPlan links a planning result to an execution plan
	LinkToExecutionPlan(ctx context.Context, planningResultID, executionPlanID string) error

	// LinkToConversation links a planning result to a conversation
	LinkToConversation(ctx context.Context, planningResultID, conversationID string) error
}
