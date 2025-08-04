package application

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	aiDomain "neuromesh/internal/ai/domain"
	"neuromesh/internal/ai/infrastructure"
	"neuromesh/internal/logging"
	"neuromesh/internal/planning/application"
)

// TestOrchestratorService_ConversationContext tests that the orchestrator
// retrieves conversation history and passes it to the planning engine
func TestOrchestratorService_ConversationContext(t *testing.T) {
	// Skip if no OpenAI API key available
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("Skipping orchestrator conversation test - OPENAI_API_KEY not set")
	}

	// ARRANGE: Set up orchestrator with conversation support
	ctx := context.Background()

	// Set up real AI provider for planning
	config := infrastructure.DefaultOpenAIConfig()
	config.APIKey = apiKey
	logger := logging.NewStructuredLogger(logging.LevelInfo)
	aiProvider := infrastructure.NewOpenAIProvider(config, logger)
	planningEngine := application.NewAIPlanningEngine(aiProvider)

	// Mock conversation service that provides conversation history
	conversationService := &MockConversationService{
		ConversationHistory: []*aiDomain.AIConversationMessage{
			aiDomain.NewAIConversationMessage("user", "I want to analyze some text"),
			aiDomain.NewAIConversationMessage("assistant", `I can help you analyze text! I have these options available:

1. **Word Count**: Count the number of words in your text
2. **Sentiment Analysis**: Determine if the text is positive, negative, or neutral

Which type of analysis would you like me to perform?`),
		},
	}

	// Mock graph explorer
	graphExplorer := &TestGraphExplorer{
		AgentContext: `{
			"available_agents": [
				{
					"id": "text-processor",
					"name": "Text Processor",
					"status": "active",
					"capabilities": [
						{
							"name": "word_count",
							"description": "Count words in text"
						},
						{
							"name": "sentiment_analysis",
							"description": "Analyze sentiment of text"
						}
					]
				}
			]
		}`,
	}

	orchestrator := NewOrchestratorService(
		planningEngine,
		graphExplorer,
		nil, // execution engine not needed for this test
		conversationService,
		nil, // repository not needed
		logger,
	)

	// Create request with conversation context
	request := &OrchestratorRequest{
		UserInput:      "Option 1 please", // Should be understood as word count from conversation context
		UserID:         "test-user-123",
		ConversationID: "test-conversation-456",
		MessageID:      "test-message-789",
	}

	// ACT: Process user request with conversation context
	// This will FAIL because orchestrator doesn't retrieve/pass conversation history yet
	result, err := orchestrator.ProcessUserRequest(ctx, request)

	// ASSERT: Should understand "Option 1" means word count from conversation context
	require.NoError(t, err, "Should process request successfully")
	require.NotNil(t, result, "Should return result")
	require.True(t, result.Success, "Should be successful")

	// Verify the planning result understands conversation context
	assert.NotNil(t, result.ExecutionPlan, "Should have execution plan")
	assert.Contains(t, result.ExecutionPlan.Intent, "word", "Should understand Option 1 as word count from conversation")
	assert.Equal(t, "EXECUTE", string(result.ExecutionPlan.Type), "Should be EXECUTE type")
}

// MockConversationService for testing conversation context
type MockConversationService struct {
	ConversationHistory []*aiDomain.AIConversationMessage
}

func (m *MockConversationService) LinkExecutionPlan(ctx context.Context, conversationID, planID string) error {
	return nil
}

func (m *MockConversationService) GetConversationHistory(ctx context.Context, conversationID string) ([]*aiDomain.AIConversationMessage, error) {
	return m.ConversationHistory, nil
}

// TestGraphExplorer for testing
type TestGraphExplorer struct {
	AgentContext string
}

func (m *TestGraphExplorer) GetAgentContext(ctx context.Context) (string, error) {
	return m.AgentContext, nil
}
