package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// RED - Write failing tests first
func TestNewConversation(t *testing.T) {
	t.Run("should create valid conversation", func(t *testing.T) {
		// Given
		id := "conv-123"
		sessionID := "session-456"
		userID := "user-789"

		// When
		conversation, err := NewConversation(id, sessionID, userID)

		// Then
		assert.NoError(t, err)
		assert.Equal(t, id, conversation.ID)
		assert.Equal(t, sessionID, conversation.SessionID)
		assert.Equal(t, userID, conversation.UserID)
		assert.Equal(t, ConversationStatusActive, conversation.Status)
		assert.Empty(t, conversation.Messages)
		assert.NotZero(t, conversation.CreatedAt)
		assert.NotZero(t, conversation.UpdatedAt)
	})

	t.Run("should fail with empty conversation ID", func(t *testing.T) {
		// When
		_, err := NewConversation("", "session-456", "user-789")

		// Then
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "conversation ID cannot be empty")
	})

	t.Run("should fail with empty session ID", func(t *testing.T) {
		// When
		_, err := NewConversation("conv-123", "", "user-789")

		// Then
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "session ID cannot be empty")
	})

	t.Run("should fail with empty user ID", func(t *testing.T) {
		// When
		_, err := NewConversation("conv-123", "session-456", "")

		// Then
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user ID cannot be empty")
	})
}

func TestConversation_AddMessage(t *testing.T) {
	t.Run("should add user message", func(t *testing.T) {
		// Given
		conversation, _ := NewConversation("conv-123", "session-456", "user-789")
		messageID := "msg-1"
		content := "Hello, count words in: Hello world"

		// When
		err := conversation.AddMessage(messageID, MessageRoleUser, content, nil)

		// Then
		assert.NoError(t, err)
		assert.Len(t, conversation.Messages, 1)

		message := conversation.Messages[0]
		assert.Equal(t, messageID, message.ID)
		assert.Equal(t, MessageRoleUser, message.Role)
		assert.Equal(t, content, message.Content)
		assert.NotZero(t, message.Timestamp)
	})

	t.Run("should add assistant message", func(t *testing.T) {
		// Given
		conversation, _ := NewConversation("conv-123", "session-456", "user-789")
		messageID := "msg-2"
		content := "The text 'Hello world' contains 2 words."

		// When
		err := conversation.AddMessage(messageID, MessageRoleAssistant, content, nil)

		// Then
		assert.NoError(t, err)
		assert.Len(t, conversation.Messages, 1)

		message := conversation.Messages[0]
		assert.Equal(t, MessageRoleAssistant, message.Role)
	})

	t.Run("should add system message", func(t *testing.T) {
		// Given
		conversation, _ := NewConversation("conv-123", "session-456", "user-789")
		messageID := "msg-3"
		content := "AI Decision: Executing word count via text-processor agent"

		// When
		err := conversation.AddMessage(messageID, MessageRoleSystem, content, map[string]interface{}{
			"decision_type": "execute",
			"agent_id":      "text-processor",
		})

		// Then
		assert.NoError(t, err)
		message := conversation.Messages[0]
		assert.Equal(t, MessageRoleSystem, message.Role)
		assert.Equal(t, "execute", message.Metadata["decision_type"])
		assert.Equal(t, "text-processor", message.Metadata["agent_id"])
	})

	t.Run("should fail with empty message ID", func(t *testing.T) {
		// Given
		conversation, _ := NewConversation("conv-123", "session-456", "user-789")

		// When
		err := conversation.AddMessage("", MessageRoleUser, "test", nil)

		// Then
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "message ID cannot be empty")
	})
}

func TestConversation_LinkExecutionPlan(t *testing.T) {
	t.Run("should link execution plan", func(t *testing.T) {
		// Given
		conversation, _ := NewConversation("conv-123", "session-456", "user-789")
		planID := "plan-abc"

		// When
		err := conversation.LinkExecutionPlan(planID)

		// Then
		assert.NoError(t, err)
		assert.Contains(t, conversation.ExecutionPlanIDs, planID)
	})

	t.Run("should fail with empty plan ID", func(t *testing.T) {
		// Given
		conversation, _ := NewConversation("conv-123", "session-456", "user-789")

		// When
		err := conversation.LinkExecutionPlan("")

		// Then
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "execution plan ID cannot be empty")
	})
}

func TestConversation_GetMessagesByRole(t *testing.T) {
	t.Run("should return messages by role", func(t *testing.T) {
		// Given
		conversation, _ := NewConversation("conv-123", "session-456", "user-789")
		conversation.AddMessage("msg-1", MessageRoleUser, "User message 1", nil)
		conversation.AddMessage("msg-2", MessageRoleAssistant, "Assistant response", nil)
		conversation.AddMessage("msg-3", MessageRoleUser, "User message 2", nil)

		// When
		userMessages := conversation.GetMessagesByRole(MessageRoleUser)
		assistantMessages := conversation.GetMessagesByRole(MessageRoleAssistant)

		// Then
		assert.Len(t, userMessages, 2)
		assert.Len(t, assistantMessages, 1)
		assert.Equal(t, "User message 1", userMessages[0].Content)
		assert.Equal(t, "User message 2", userMessages[1].Content)
		assert.Equal(t, "Assistant response", assistantMessages[0].Content)
	})
}
