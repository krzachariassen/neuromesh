package application

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	aiDomain "neuromesh/internal/ai/domain"
	"neuromesh/internal/ai/infrastructure"
	executionDomain "neuromesh/internal/execution/domain"
	"neuromesh/internal/logging"
	planningDomain "neuromesh/internal/planning/domain"
	"neuromesh/testHelpers"
)

// TestConversationSummarizer_TDD_Basic follows TDD methodology with basic functionality
func TestConversationSummarizer_TDD_Basic(t *testing.T) {
	// RED: Test fails because summarizer doesn't exist yet
	t.Run("RED: should require AI provider", func(t *testing.T) {
		summarizer := NewConversationSummarizer(nil, nil, nil)

		ctx := context.Background()
		_, err := summarizer.SummarizeConversation(ctx, "conv-123", "plan-123")

		assert.Error(t, err, "Should fail without AI provider")
		assert.Contains(t, err.Error(), "aiProvider is required")
	})

	t.Run("RED: should require repository", func(t *testing.T) {
		// Create basic AI provider for testing
		logger := logging.NewStructuredLogger(logging.LevelInfo)
		aiConfig := &infrastructure.OpenAIConfig{
			APIKey:    "fake-key",
			Model:     "gpt-4o-mini",
			BaseURL:   "https://api.openai.com/v1",
			MaxTokens: 1000,
		}
		aiProvider := infrastructure.NewOpenAIProvider(aiConfig, logger)
		summarizer := NewConversationSummarizer(aiProvider, nil, nil)

		ctx := context.Background()
		_, err := summarizer.SummarizeConversation(ctx, "conv-123", "plan-123")

		assert.Error(t, err, "Should fail without repository")
		assert.Contains(t, err.Error(), "repository is required")
	})

	t.Run("GREEN: should validate input parameters", func(t *testing.T) {
		// Test parameter validation
		logger := logging.NewStructuredLogger(logging.LevelInfo)
		aiConfig := &infrastructure.OpenAIConfig{
			APIKey:    "fake-key",
			Model:     "gpt-4o-mini",
			BaseURL:   "https://api.openai.com/v1",
			MaxTokens: 1000,
		}
		aiProvider := infrastructure.NewOpenAIProvider(aiConfig, logger)
		mockRepo := testHelpers.NewMockExecutionPlanRepository()
		summarizer := NewConversationSummarizer(aiProvider, mockRepo, nil)

		ctx := context.Background()

		// Test empty conversation ID
		_, err := summarizer.SummarizeConversation(ctx, "", "plan-123")
		assert.Error(t, err, "Should fail with empty conversation ID")

		// Test empty plan ID
		_, err = summarizer.SummarizeConversation(ctx, "conv-123", "")
		assert.Error(t, err, "Should fail with empty plan ID")
	})
}

// TestConversationSummarizer_RealAI tests with REAL AI following "never mock AI" principle
func TestConversationSummarizer_RealAI_Basic(t *testing.T) {
	// Skip if no OpenAI API key (CI/CD environments)
	if os.Getenv("OPENAI_API_KEY") == "" {
		t.Skip("Skipping real AI test - OPENAI_API_KEY not set")
	}

	t.Run("GREEN: should create summary with real AI when results available", func(t *testing.T) {
		// Setup REAL AI provider (never mock AI per instructions)
		logger := logging.NewStructuredLogger(logging.LevelInfo)
		aiConfig := &infrastructure.OpenAIConfig{
			APIKey:    os.Getenv("OPENAI_API_KEY"),
			Model:     "gpt-4o-mini", // Use cost-effective model for testing
			BaseURL:   "https://api.openai.com/v1",
			MaxTokens: 1000,
		}
		aiProvider := infrastructure.NewOpenAIProvider(aiConfig, logger)

		// Mock repository with proper plan-step-result hierarchy
		mockRepo := testHelpers.NewMockExecutionPlanRepository()
		mockConversationService := testHelpers.NewMockConversationService()

		// Setup conversation history mock for conversation-aware summarization
		conversationHistory := []*aiDomain.AIConversationMessage{
			{Role: "user", Content: "Count words in 'hello world'"},
			{Role: "assistant", Content: "I'll count the words in that text for you."},
		}
		mockConversationService.On("GetConversationHistory", mock.Anything, "test-conversation").Return(conversationHistory, nil)

		summarizer := NewConversationSummarizer(aiProvider, mockRepo, mockConversationService)

		// Create execution plan with steps (using planning domain as expected by mock)
		planID := "word-count-plan-123"

		// Add step to plan (this creates the step association)
		stepID := "step-1"
		err := mockRepo.AddStep(context.Background(), &planningDomain.ExecutionStep{
			ID:            stepID,
			PlanID:        planID, // Associate with plan
			Name:          "Count words",
			Description:   "Count words in text",
			AssignedAgent: "text-processor",
		})
		require.NoError(t, err)

		// Now store agent result linked to the step
		agentResult := &executionDomain.AgentResult{
			ID:              "result-1",
			ExecutionStepID: stepID, // Link to the step
			AgentID:         "text-processor",
			Content:         "The text 'hello world' contains 2 words.",
			Status:          executionDomain.AgentResultStatusSuccess,
		}
		err = mockRepo.StoreAgentResult(context.Background(), agentResult)
		require.NoError(t, err)

		// ACT: Create conversation summary with REAL AI
		summary, err := summarizer.SummarizeConversation(context.Background(), "test-conversation", planID)

		// ASSERT: Verify real AI created proper summary
		require.NoError(t, err, "Should create summary successfully")
		require.NotNil(t, summary, "Should return summary")

		// Verify summary structure
		assert.NotEmpty(t, summary.ID, "Should have summary ID")
		assert.Equal(t, "test-conversation", summary.ConversationID)
		assert.Equal(t, planID, summary.PlanID)
		assert.NotEmpty(t, summary.Summary, "Should have AI-generated summary")

		t.Logf("AI Generated Summary: %s", summary.Summary)
		t.Logf("User Result: %s", summary.UserResult)

		// Verify AI markers are present (JSON format)
		assert.Contains(t, summary.Summary, "user_answer", "Should contain user_answer JSON field")
		assert.Contains(t, summary.Summary, "conversation_summary", "Should contain conversation_summary JSON field")

		// Verify extraction works
		userContent, err := executionDomain.ExtractUserFriendlyContent(summary.Summary)
		assert.NoError(t, err, "Should extract user-friendly content")
		assert.NotNil(t, userContent, "Should extract user-friendly content")
		assert.NotEmpty(t, userContent.Answer, "Should have extracted answer")

		t.Logf("✅ REAL AI Summary created with markers:")
		t.Logf("Summary ID: %s", summary.ID)
		t.Logf("Raw summary: %s", summary.Summary)
		t.Logf("Extracted answer: %s", userContent.Answer)
		t.Logf("User result: %s", summary.UserResult)
	})
}

// TestConversationSummary_DomainValidation tests the domain model separately
func TestConversationSummary_DomainValidation(t *testing.T) {
	t.Run("GREEN: should create valid conversation summary", func(t *testing.T) {
		summary, err := executionDomain.NewConversationSummary(
			"conv-123",
			"plan-123",
			"Full conversation summary with details",
			"Simple answer for user",
		)

		assert.NoError(t, err)
		assert.NotNil(t, summary)
		assert.NotEmpty(t, summary.ID)
		assert.Equal(t, "conv-123", summary.ConversationID)
		assert.Equal(t, "plan-123", summary.PlanID)
		assert.Equal(t, "Full conversation summary with details", summary.Summary)
		assert.Equal(t, "Simple answer for user", summary.UserResult)
	})

	t.Run("RED: should validate required fields", func(t *testing.T) {
		// Test empty conversation ID
		_, err := executionDomain.NewConversationSummary("", "plan-123", "summary", "result")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "conversation ID cannot be empty")

		// Test empty plan ID
		_, err = executionDomain.NewConversationSummary("conv-123", "", "summary", "result")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "plan ID cannot be empty")

		// Test empty user result
		_, err = executionDomain.NewConversationSummary("conv-123", "plan-123", "summary", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user result cannot be empty")
	})
}
