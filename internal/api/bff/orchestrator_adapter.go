package bff

import (
	"context"

	"neuromesh/internal/orchestrator/application"
)

// OrchestratorAdapter adapts the new clean architecture orchestrator
// to the BFF interface expectations
type OrchestratorAdapter struct {
	orchestratorService *application.OrchestratorService
}

// NewOrchestratorAdapter creates a new adapter
func NewOrchestratorAdapter(orchestratorService *application.OrchestratorService) *OrchestratorAdapter {
	return &OrchestratorAdapter{
		orchestratorService: orchestratorService,
	}
}

// ProcessRequest adapts the new ProcessUserRequest to the BFF interface
func (a *OrchestratorAdapter) ProcessRequest(ctx context.Context, userInput, userID string) (*application.OrchestratorResult, error) {
	request := &application.OrchestratorRequest{
		UserInput: userInput,
		UserID:    userID,
	}

	result, err := a.orchestratorService.ProcessUserRequest(ctx, request)
	if err != nil {
		return nil, err
	}

	// Return the result directly - no more conversion needed!
	return result, nil
}

// ProcessUserRequest implements the BFF interface directly
func (a *OrchestratorAdapter) ProcessUserRequest(ctx context.Context, request *application.OrchestratorRequest) (*application.OrchestratorResult, error) {
	return a.orchestratorService.ProcessUserRequest(ctx, request)
}
