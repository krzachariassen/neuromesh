package domain

import (
	"context"
)

// AnalysisRepository defines the repository interface for Analysis persistence
type AnalysisRepository interface {
	// Store persists an Analysis in the graph with proper relationships
	Store(ctx context.Context, analysis *Analysis) error

	// GetByID retrieves an Analysis by its ID
	GetByID(ctx context.Context, analysisID string) (*Analysis, error)

	// GetByRequestID retrieves an Analysis by the request (message) ID
	GetByRequestID(ctx context.Context, requestID string) (*Analysis, error)

	// GetByUserID retrieves all analyses for a specific user, ordered by timestamp desc
	GetByUserID(ctx context.Context, userID string, limit int) ([]*Analysis, error)

	// GetByConfidenceRange retrieves analyses within a confidence range
	GetByConfidenceRange(ctx context.Context, minConfidence, maxConfidence int, limit int) ([]*Analysis, error)

	// GetByCategory retrieves analyses by category
	GetByCategory(ctx context.Context, category string, limit int) ([]*Analysis, error)
}
