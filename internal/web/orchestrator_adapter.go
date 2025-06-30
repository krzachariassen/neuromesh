package web

import (
	"context"

	"neuromesh/internal/orchestrator/application"
)

// OrchestratorAdapter adapts the new clean architecture orchestrator
// to the web interface expectations
type OrchestratorAdapter struct {
	orchestratorService *application.OrchestratorService
}

// NewOrchestratorAdapter creates a new adapter
func NewOrchestratorAdapter(orchestratorService *application.OrchestratorService) *OrchestratorAdapter {
	return &OrchestratorAdapter{
		orchestratorService: orchestratorService,
	}
}

// ProcessRequest adapts the new ProcessUserRequest to the web interface
func (w *OrchestratorAdapter) ProcessRequest(ctx context.Context, userInput, userID string) (*application.OrchestratorResult, error) {
	request := &application.OrchestratorRequest{
		UserInput: userInput,
		UserID:    userID,
	}

	result, err := w.orchestratorService.ProcessUserRequest(ctx, request)
	if err != nil {
		return nil, err
	}

	// Return the result directly - no more conversion needed!
	return result, nil
}
