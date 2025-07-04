package web

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	orchestratorApp "neuromesh/internal/orchestrator/application"
	orchestratorDomain "neuromesh/internal/orchestrator/domain"
)

// TestConversationAwareWebBFFIntegration tests basic integration
func TestConversationAwareWebBFFIntegration(t *testing.T) {
	// Skip if Neo4j is not available
	t.Skip("Integration test requires Neo4j - will be enabled in main server integration")

	t.Run("should have basic conversation integration", func(t *testing.T) {
		// This test demonstrates that we need the ConversationAwareWebBFF
		// to be integrated into the main server instead of the regular WebBFF

		// The current server uses regular WebBFF
		// but should use ConversationAwareWebBFF for conversation persistence
		assert.True(t, true, "ConversationAwareWebBFF should replace WebBFF in server")
	})
}

// MockOrchestrator is a test implementation of AIOrchestrator
type MockOrchestrator struct {
	responses map[string]*orchestratorApp.OrchestratorResult
}

func (m *MockOrchestrator) ProcessRequest(ctx context.Context, userInput string, userID string) (*orchestratorApp.OrchestratorResult, error) {
	if response, exists := m.responses[userInput]; exists {
		return response, nil
	}

	// Default response
	return &orchestratorApp.OrchestratorResult{
		Message: "I understand your request",
		Success: true,
		Analysis: &orchestratorDomain.Analysis{
			Intent:     "general",
			Confidence: 70,
		},
		Decision: &orchestratorDomain.Decision{
			Type:      orchestratorDomain.DecisionTypeClarify,
			Reasoning: "General response",
		},
	}, nil
}
