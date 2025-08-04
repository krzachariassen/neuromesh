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
	"neuromesh/internal/planning/domain"
)

// TestCreateExecutionPlan_WithConversationContext tests that the planning engine
// can use conversation context to understand follow-up requests like "Option 1"
func TestCreateExecutionPlan_WithConversationContext(t *testing.T) {
	// Skip if no OpenAI API key available
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("Skipping conversation context test - OPENAI_API_KEY not set")
	}

	// ARRANGE: Set up AI provider and planning engine
	config := infrastructure.DefaultOpenAIConfig()
	config.APIKey = apiKey
	logger := logging.NewStructuredLogger(logging.LevelInfo)
	aiProvider := infrastructure.NewOpenAIProvider(config, logger)
	engine := NewAIPlanningEngine(aiProvider)

	ctx := context.Background()
	userID := "test-user-123"
	requestID := "test-request-456"

	// Agent context with text processing capabilities
	agentContext := `{
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
	}`

	// Simulate conversation history where AI offered clarification options
	conversationHistory := []*aiDomain.AIConversationMessage{
		aiDomain.NewAIConversationMessage("system", "You are an AI planning orchestrator..."),
		aiDomain.NewAIConversationMessage("user", "I want to analyze some text"),
		aiDomain.NewAIConversationMessage("assistant", `I can help you analyze text! I have these options available:

1. **Word Count**: Count the number of words in your text
2. **Sentiment Analysis**: Determine if the text is positive, negative, or neutral

Which type of analysis would you like me to perform? Please let me know and provide your text.`),
		aiDomain.NewAIConversationMessage("user", "Option 1 please"), // This should be understood as word count
	}

	// ACT: Create execution plan with conversation context
	// This will FAIL because CreateExecutionPlan doesn't support conversation history yet
	plan, err := engine.CreateExecutionPlan(ctx, "Option 1 please", userID, agentContext, requestID, conversationHistory)

	// ASSERT: Planning should understand "Option 1" means word count from conversation context
	require.NoError(t, err, "Should create execution plan successfully")
	require.NotNil(t, plan, "Should return execution plan")

	// Verify plan understands the conversation context
	assert.Contains(t, plan.Intent, "word", "Intent should reference word counting based on conversation context")
	assert.Equal(t, domain.PlanningTypeExecute, plan.Type, "Should be EXECUTE type since capability is available")

	// Verify execution steps use the correct agent
	require.Len(t, plan.Steps, 1, "Should have one execution step")
	assert.Equal(t, "text-processor", plan.Steps[0].AssignedAgent, "Should use text-processor agent")
	assert.Contains(t, plan.Steps[0].Description, "word", "Step should involve word counting")
}

// TestCreateExecutionPlan_BackwardCompatibility ensures existing calls without conversation still work
func TestCreateExecutionPlan_BackwardCompatibility(t *testing.T) {
	// Skip if no OpenAI API key available
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("Skipping backward compatibility test - OPENAI_API_KEY not set")
	}

	// ARRANGE: Set up AI provider and planning engine
	config := infrastructure.DefaultOpenAIConfig()
	config.APIKey = apiKey
	logger := logging.NewStructuredLogger(logging.LevelInfo)
	aiProvider := infrastructure.NewOpenAIProvider(config, logger)
	engine := NewAIPlanningEngine(aiProvider)

	ctx := context.Background()
	userID := "test-user-123"
	requestID := "test-request-456"

	agentContext := `{
		"available_agents": [
			{
				"id": "text-processor",
				"name": "Text Processor", 
				"status": "active",
				"capabilities": [
					{
						"name": "word_count",
						"description": "Count words in text"
					}
				]
			}
		]
	}`

	// ACT: Call with existing signature (no conversation history)
	plan, err := engine.CreateExecutionPlan(ctx, "Count words in my text", userID, agentContext, requestID)

	// ASSERT: Should work exactly as before
	require.NoError(t, err, "Should create execution plan successfully")
	require.NotNil(t, plan, "Should return execution plan")
	assert.Contains(t, plan.Intent, "count", "Should understand word counting intent")
	assert.Equal(t, domain.PlanningTypeExecute, plan.Type, "Should be EXECUTE type")
}

// TestCreateExecutionPlan_ConversationClarification tests that conversation context helps with clarification
func TestCreateExecutionPlan_ConversationClarification(t *testing.T) {
	// Skip if no OpenAI API key available
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("Skipping conversation clarification test - OPENAI_API_KEY not set")
	}

	// ARRANGE: Set up AI provider and planning engine
	config := infrastructure.DefaultOpenAIConfig()
	config.APIKey = apiKey
	logger := logging.NewStructuredLogger(logging.LevelInfo)
	aiProvider := infrastructure.NewOpenAIProvider(config, logger)
	engine := NewAIPlanningEngine(aiProvider)

	ctx := context.Background()
	userID := "test-user-123"
	requestID := "test-request-456"

	// Limited agent context - no translation capability
	agentContext := `{
		"available_agents": [
			{
				"id": "text-processor",
				"name": "Text Processor",
				"status": "active", 
				"capabilities": [
					{
						"name": "word_count",
						"description": "Count words in text"
					}
				]
			}
		]
	}`

	// Conversation where user asked for something we can't do
	conversationHistory := []*aiDomain.AIConversationMessage{
		aiDomain.NewAIConversationMessage("user", "I need to translate text from English to Spanish"),
		aiDomain.NewAIConversationMessage("assistant", "I don't have translation capabilities available. I can offer:\n\n1. Word count analysis\n\nWould you like me to count words instead?"),
		aiDomain.NewAIConversationMessage("user", "What other options do I have?"),
	}

	// ACT: Create execution plan with conversation context
	// This will FAIL because CreateExecutionPlan doesn't support conversation history yet
	plan, err := engine.CreateExecutionPlan(ctx, "What other options do I have?", userID, agentContext, requestID, conversationHistory)

	// ASSERT: Should provide appropriate response based on conversation context
	require.NoError(t, err, "Should create execution plan successfully")
	require.NotNil(t, plan, "Should return execution plan")

	// Should understand this is about exploring available options given the conversation
	// Either CLARIFY (for more options) or EXECUTE (with available word count) are both valid
	assert.Contains(t, []string{"CLARIFY", "EXECUTE"}, string(plan.Type),
		"Should be CLARIFY (for more options) or EXECUTE (with available capability)")

	// If CLARIFY, should mention available capabilities
	if plan.Type == "CLARIFY" {
		assert.Contains(t, plan.Description, "word", "Clarification should mention available word count capability")
	}

	// If EXECUTE, should use available agent
	if plan.Type == "EXECUTE" {
		require.Len(t, plan.Steps, 1, "Should have one execution step")
		assert.Equal(t, "text-processor", plan.Steps[0].AssignedAgent, "Should use text-processor agent")
	}
}
