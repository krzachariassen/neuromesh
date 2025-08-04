package agent

import (
	"testing"

	pb "github.com/ztdp/agents/text-translator/proto/api"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAINativeAgent_ProcessInstruction(t *testing.T) {
	// Create an agent for testing
	config := Config{
		AgentID:             "test-agent",
		Name:                "Test Agent",
		OrchestratorAddress: "localhost:50051",
	}
	agent := NewAINativeAgent(config)

	t.Run("should translate text correctly", func(t *testing.T) {
		testCases := []struct {
			name        string
			instruction string
			expected    string
		}{
			{
				name:        "translate to spanish with quotes",
				instruction: `Translate "hello world" to Spanish`,
				expected:    `Translation to Spanish: "Hola mundo"`,
			},
			{
				name:        "translate to french with single quotes",
				instruction: `Translate 'the quick brown fox' to French`,
				expected:    `Translation to French: "le renard brun rapide"`,
			},
			{
				name:        "translate with following pattern",
				instruction: `Translate the following text to German: the quick brown fox`,
				expected:    `Translation to German: "der schnelle braune Fuchs"`,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result := agent.ProcessInstruction(tc.instruction)
				assert.Equal(t, tc.expected, result)
			})
		}
	})

	t.Run("should detect language correctly", func(t *testing.T) {
		instruction := `Detect the language of: "Hola mundo"`
		result := agent.ProcessInstruction(instruction)

		// Should contain language detection information
		assert.Equal(t, "Detected language: Spanish", result)
	})

	t.Run("should format translation correctly", func(t *testing.T) {
		instruction := `Format translation of "hello world" to Portuguese`
		result := agent.ProcessInstruction(instruction)

		assert.Contains(t, result, "Formatted translation:")
		assert.Contains(t, result, "hello world")
		assert.Contains(t, result, "Portuguese")
	})

	t.Run("should default to english translation for unclear instructions", func(t *testing.T) {
		instruction := `Process this text: "hello world"`
		result := agent.ProcessInstruction(instruction)

		assert.Contains(t, result, "Translation to English:")
		assert.Contains(t, result, "[English] hello world")
	})

	t.Run("should handle conversation stream messages", func(t *testing.T) {
		// Test that the agent can process instruction messages from a conversation stream
		// This tests the integration between stream message handling and instruction processing

		instruction := `Translate "hello world" to Spanish`
		expectedContent := `Translation to Spanish: "Hola mundo"`

		// Create a mock conversation message
		msg := &pb.ConversationMessage{
			MessageId:     "test-msg-1",
			CorrelationId: "test-corr-1",
			FromId:        "orchestrator",
			ToId:          agent.config.AgentID,
			Type:          pb.MessageType_MESSAGE_TYPE_INSTRUCTION,
			Content:       instruction,
			Context:       nil,
		}

		// Process the message (this should call ProcessInstruction internally)
		response := agent.processConversationMessage(msg)

		// Verify the response is a completion message
		assert.NotNil(t, response)
		assert.Equal(t, pb.MessageType_MESSAGE_TYPE_COMPLETION, response.Type)
		assert.Equal(t, agent.config.AgentID, response.FromId)
		assert.Equal(t, "orchestrator", response.ToId)
		assert.Equal(t, "test-corr-1", response.CorrelationId)
		assert.Equal(t, expectedContent, response.Content)
	})
}

func TestAINativeAgent_ExtractTextFromInstruction(t *testing.T) {
	config := Config{
		AgentID:             "test-agent",
		Name:                "Test Agent",
		OrchestratorAddress: "localhost:50051",
	}
	agent := NewAINativeAgent(config)

	testCases := []struct {
		name        string
		instruction string
		expected    string
	}{
		{
			name:        "double quotes",
			instruction: `Count words in "Hello world"`,
			expected:    "Hello world",
		},
		{
			name:        "single quotes",
			instruction: `Analyze 'Beautiful day'`,
			expected:    "Beautiful day",
		},
		{
			name:        "text colon pattern",
			instruction: `Process text: This is a test`,
			expected:    "This is a test",
		},
		{
			name:        "following pattern",
			instruction: `Count the words in the following: Quick brown fox`,
			expected:    "Quick brown fox",
		},
		{
			name:        "in pattern",
			instruction: `Count words in Beautiful day today`,
			expected:    "Beautiful day today",
		},
		{
			name:        "fallback to entire instruction",
			instruction: `Just some text here`,
			expected:    "Just some text here",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := agent.extractTextFromInstruction(tc.instruction)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestAINativeAgent_TranslateText(t *testing.T) {
	config := Config{
		AgentID:             "test-agent",
		Name:                "Test Agent",
		OrchestratorAddress: "localhost:50051",
	}
	agent := NewAINativeAgent(config)

	testCases := []struct {
		name       string
		text       string
		targetLang string
		expected   string
	}{
		{
			name:       "spanish translation",
			text:       "hello world",
			targetLang: "Spanish",
			expected:   "Hola mundo",
		},
		{
			name:       "french translation",
			text:       "hello world",
			targetLang: "French",
			expected:   "Bonjour le monde",
		},
		{
			name:       "empty text",
			text:       "",
			targetLang: "Spanish",
			expected:   "",
		},
		{
			name:       "german translation",
			text:       "the quick brown fox",
			targetLang: "German",
			expected:   "der schnelle braune Fuchs",
		},
		{
			name:       "portuguese translation",
			text:       "hello world",
			targetLang: "Portuguese",
			expected:   "Olá mundo",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := agent.translateText(tc.text, tc.targetLang)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestAINativeAgent_DetectLanguage(t *testing.T) {
	config := Config{
		AgentID:             "test-agent",
		Name:                "Test Agent",
		OrchestratorAddress: "localhost:50051",
	}
	agent := NewAINativeAgent(config)

	testCases := []struct {
		name     string
		text     string
		expected string
	}{
		{
			name:     "english text",
			text:     "Hello world",
			expected: "English",
		},
		{
			name:     "spanish text",
			text:     "Hola mundo",
			expected: "Spanish",
		},
		{
			name:     "french text",
			text:     "Bonjour le monde",
			expected: "French",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := agent.detectLanguage(tc.text)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestAINativeAgent_GetCapabilities(t *testing.T) {
	config := Config{
		AgentID:             "test-agent",
		Name:                "Test Agent",
		OrchestratorAddress: "localhost:50051",
	}
	agent := NewAINativeAgent(config)

	capabilities := agent.getCapabilities()

	require.Len(t, capabilities, 3)

	// Check that we have the expected capabilities
	capabilityNames := make([]string, len(capabilities))
	for i, cap := range capabilities {
		capabilityNames[i] = cap.Name
	}

	assert.Contains(t, capabilityNames, "translate-text")
	assert.Contains(t, capabilityNames, "detect-language")
	assert.Contains(t, capabilityNames, "format-translation")

	// Check descriptions are present
	for _, cap := range capabilities {
		assert.NotEmpty(t, cap.Description)
		assert.NotEmpty(t, cap.Inputs)
		assert.NotEmpty(t, cap.Outputs)
	}
}

func TestNewAINativeAgent(t *testing.T) {
	config := Config{
		AgentID:             "test-agent-123",
		Name:                "Test Agent",
		OrchestratorAddress: "localhost:50051",
	}

	agent := NewAINativeAgent(config)

	assert.NotNil(t, agent)
	assert.Equal(t, config.AgentID, agent.config.AgentID)
	assert.Equal(t, config.Name, agent.config.Name)
	assert.Equal(t, config.OrchestratorAddress, agent.config.OrchestratorAddress)
	assert.False(t, agent.registered)
	assert.Empty(t, agent.sessionID)
}

// func TestAINativeAgent_HeartbeatInterval(t *testing.T) {
// 	// Arrange
// 	agent := NewAINativeAgent(Config{
// 		AgentID:             "test-interval-agent",
// 		Name:                "Test Interval Agent",
// 		OrchestratorAddress: "localhost:50051",
// 		ReconnectInterval:   time.Second,
// 	})
//
// 	ctx, cancel := context.WithTimeout(context.Background(), 95*time.Second)
// 	defer cancel()
//
// 	// TDD RED: This should fail as method doesn't exist
// 	heartbeatSent := make(chan bool, 5) // Buffer for multiple heartbeats
// 	err := agent.StartHeartbeat(ctx, heartbeatSent)
// 	require.NoError(t, err)
//
// 	// Count heartbeats over 90 seconds (should get at least 3 heartbeats)
// 	heartbeatCount := 0
// 	timeout := time.After(90 * time.Second)
//
// 	for heartbeatCount < 3 {
// 		select {
// 		case <-heartbeatSent:
// 			heartbeatCount++
// 		case <-timeout:
// 			t.Fatalf("Expected at least 3 heartbeats in 90 seconds, got %d", heartbeatCount)
// 		}
// 	}
//
// 	assert.GreaterOrEqual(t, heartbeatCount, 3, "Should receive at least 3 heartbeats in 90 seconds")
// }
