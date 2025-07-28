package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"neuromesh/testHelpers"
)

// RED Phase: Write failing test that exposes missing repository integration
func TestAIPlanningEngine_CreateExecutionPlan_WithPlanningRepositoryIntegration(t *testing.T) {
	t.Run("should store planning result when repository is injected", func(t *testing.T) {
		// Arrange: Set up with real AI provider and mock planning repository
		ctx := context.Background()
		aiProvider := testHelpers.SetupRealAIProvider(t)

		// Create mock planning repository to capture the stored result
		mockPlanningRepo := &testHelpers.MockPlanningResultRepository{}
		mockExecutionRepo := testHelpers.NewMockExecutionPlanRepository()

		// Create engine with both repositories
		engine := NewAIPlanningEngineWithRepositories(aiProvider, mockExecutionRepo, mockPlanningRepo)

		userID := "test-user-123"
		requestID := "test-request-456"
		userInput := "Count the words in this text: Hello world"
		agentContext := `Available Agents:
- text-processor | Status: available | Capabilities: word count, text analysis, character count`

		// Act: Create execution plan
		result, err := engine.CreateExecutionPlan(ctx, userInput, userID, agentContext, requestID)

		// Assert: Should return valid planning result
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, requestID, result.RequestID)

		// CRITICAL: Planning result should be stored in repository
		assert.True(t, mockPlanningRepo.StoreCalled, "Planning result should be stored in repository")
		assert.Equal(t, result, mockPlanningRepo.StoredResult, "Stored result should match returned result")

		// Should also be linked to request
		assert.True(t, mockPlanningRepo.LinkToRequestCalled, "Planning result should be linked to request")
		assert.Equal(t, result.ID, mockPlanningRepo.LinkedPlanningResultID)
		assert.Equal(t, requestID, mockPlanningRepo.LinkedRequestID)

		t.Logf("✅ Planning result stored with ID: %s", result.ID)
		t.Logf("✅ Planning result linked to request: %s", requestID)
	})

	t.Run("should handle planning repository storage errors gracefully", func(t *testing.T) {
		// Arrange: Set up with failing repository
		ctx := context.Background()
		aiProvider := testHelpers.SetupRealAIProvider(t)

		failingRepo := &testHelpers.MockPlanningResultRepository{
			ShouldFailStore: true,
		}
		mockExecutionRepo := testHelpers.NewMockExecutionPlanRepository()

		engine := NewAIPlanningEngineWithRepositories(aiProvider, mockExecutionRepo, failingRepo)

		userID := "test-user-123"
		requestID := "test-request-456"
		userInput := "Count the words in this text: Hello world"
		agentContext := `Available Agents:
- text-processor | Status: available | Capabilities: word count, text analysis`

		// Act & Assert: Should handle repository error gracefully
		result, err := engine.CreateExecutionPlan(ctx, userInput, userID, agentContext, requestID)

		// Should return error due to repository failure
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to store planning result")
	})
}
