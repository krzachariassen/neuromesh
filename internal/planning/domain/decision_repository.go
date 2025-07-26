package domain

import (
	"context"
)

// DecisionRepository defines the repository interface for Decision persistence
type DecisionRepository interface {
	// Store persists a Decision in the graph with proper relationships
	Store(ctx context.Context, decision *Decision) error

	// GetByID retrieves a Decision by its ID
	GetByID(ctx context.Context, decisionID string) (*Decision, error)

	// GetByRequestID retrieves a Decision by the request (message) ID
	GetByRequestID(ctx context.Context, requestID string) (*Decision, error)

	// GetByAnalysisID retrieves a Decision by the analysis ID
	GetByAnalysisID(ctx context.Context, analysisID string) (*Decision, error)

	// GetByUserID retrieves all decisions for a specific user, ordered by timestamp desc
	GetByUserID(ctx context.Context, userID string, limit int) ([]*Decision, error)

	// GetByType retrieves decisions by type (CLARIFY or EXECUTE)
	GetByType(ctx context.Context, decisionType DecisionType, limit int) ([]*Decision, error)

	// LinkToAnalysis creates a relationship between decision and analysis
	LinkToAnalysis(ctx context.Context, decisionID, analysisID string) error

	// LinkToExecutionPlan creates a relationship between decision and execution plan
	LinkToExecutionPlan(ctx context.Context, decisionID, executionPlanID string) error
}
