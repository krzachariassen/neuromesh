package agent

import (
	"context"
	"testing"
	"time"

	pb "github.com/ztdp/agents/text-processor/proto/api"

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

	t.Run("should count words correctly", func(t *testing.T) {
		testCases := []struct {
			name        string
			instruction string
			expected    string
		}{
			{
				name:        "simple word count with quotes",
				instruction: `Count the number of words in "Hello world"`,
				expected:    `The text "Hello world" contains 2 words.`,
			},
			{
				name:        "word count with single quotes",
				instruction: `Count words in 'This is a test'`,
				expected:    `The text "This is a test" contains 4 words.`,
			},
			{
				name:        "word count with following pattern",
				instruction: `Count the words in the following text: Beautiful day today`,
				expected:    `The text "Beautiful day today" contains 3 words.`,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result := agent.ProcessInstruction(tc.instruction)
				assert.Equal(t, tc.expected, result)
			})
		}
	})

	t.Run("should analyze text correctly", func(t *testing.T) {
		instruction := `Analyze the text: "Hello world"`
		result := agent.ProcessInstruction(instruction)

		// Should contain analysis information
		assert.Contains(t, result, "Hello world")
		assert.Contains(t, result, "2 words")
		assert.Contains(t, result, "11 characters")
		assert.Contains(t, result, "10 letters")
	})

	t.Run("should count characters correctly", func(t *testing.T) {
		instruction := `Count characters in "Hello"`
		result := agent.ProcessInstruction(instruction)

		assert.Equal(t, `The text "Hello" contains 5 characters.`, result)
	})

	t.Run("should default to word count for unclear instructions", func(t *testing.T) {
		instruction := `Process this text: "Default test"`
		result := agent.ProcessInstruction(instruction)

		assert.Contains(t, result, "2 words")
	})

	t.Run("should handle conversation stream messages", func(t *testing.T) {
		// Test that the agent can process instruction messages from a conversation stream
		// This tests the integration between stream message handling and instruction processing

		instruction := `Count the words in "Hello world"`
		expectedContent := `The text "Hello world" contains 2 words.`

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

func TestAINativeAgent_CountWords(t *testing.T) {
	config := Config{
		AgentID:             "test-agent",
		Name:                "Test Agent",
		OrchestratorAddress: "localhost:50051",
	}
	agent := NewAINativeAgent(config)

	testCases := []struct {
		name     string
		text     string
		expected int
	}{
		{
			name:     "simple text",
			text:     "Hello world",
			expected: 2,
		},
		{
			name:     "multiple spaces",
			text:     "Hello    world   test",
			expected: 3,
		},
		{
			name:     "empty text",
			text:     "",
			expected: 0,
		},
		{
			name:     "whitespace only",
			text:     "   ",
			expected: 0,
		},
		{
			name:     "single word",
			text:     "Hello",
			expected: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := agent.countWords(tc.text)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestAINativeAgent_AnalyzeText(t *testing.T) {
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
			name:     "simple text",
			text:     "Hello",
			expected: "1 words, 5 characters, 5 letters",
		},
		{
			name:     "text with spaces and punctuation",
			text:     "Hello, world!",
			expected: "2 words, 13 characters, 10 letters",
		},
		{
			name:     "empty text",
			text:     "",
			expected: "empty text",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := agent.analyzeText(tc.text)
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

	assert.Contains(t, capabilityNames, "word-count")
	assert.Contains(t, capabilityNames, "text-analysis")
	assert.Contains(t, capabilityNames, "character-count")

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

// TDD RED: Test for agent heartbeat functionality
func TestAINativeAgent_StartHeartbeat(t *testing.T) {
	// Arrange
	agent := NewAINativeAgent(Config{
		AgentID:             "test-heartbeat-agent",
		Name:                "Test Heartbeat Agent",
		OrchestratorAddress: "localhost:50051",
		ReconnectInterval:   time.Second,
	})

	// Create a context that will be cancelled after testing
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// TDD RED: This method doesn't exist yet, should fail
	heartbeatSent := make(chan bool, 1)
	err := agent.StartHeartbeat(ctx, heartbeatSent)

	// Assert
	assert.NoError(t, err, "StartHeartbeat should not return an error")

	// Wait for at least one heartbeat to be sent
	select {
	case <-heartbeatSent:
		// Success - heartbeat was sent
	case <-time.After(35 * time.Second):
		t.Fatal("No heartbeat was sent within 35 seconds")
	}
}

func TestAINativeAgent_HeartbeatInterval(t *testing.T) {
	// Arrange
	agent := NewAINativeAgent(Config{
		AgentID:             "test-interval-agent",
		Name:                "Test Interval Agent",
		OrchestratorAddress: "localhost:50051",
		ReconnectInterval:   time.Second,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 95*time.Second)
	defer cancel()

	// TDD RED: This should fail as method doesn't exist
	heartbeatSent := make(chan bool, 5) // Buffer for multiple heartbeats
	err := agent.StartHeartbeat(ctx, heartbeatSent)
	require.NoError(t, err)

	// Count heartbeats over 90 seconds (should get at least 3 heartbeats)
	heartbeatCount := 0
	timeout := time.After(90 * time.Second)

	for heartbeatCount < 3 {
		select {
		case <-heartbeatSent:
			heartbeatCount++
		case <-timeout:
			t.Fatalf("Expected at least 3 heartbeats in 90 seconds, got %d", heartbeatCount)
		}
	}

	assert.GreaterOrEqual(t, heartbeatCount, 3, "Should receive at least 3 heartbeats in 90 seconds")
}
