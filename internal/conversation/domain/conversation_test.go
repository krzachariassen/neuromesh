package domain

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Graph-Native TDD Tests - Updated for conversation entity without embedded foreign keys
func TestNewConversation(t *testing.T) {
	t.Run("should create valid conversation with graph-native structure", func(t *testing.T) {
		// Given
		id := "conv-123"

		// When
		conversation, err := NewConversation(id)

		// Then
		assert.NoError(t, err)
		assert.Equal(t, id, conversation.ID)
		assert.Equal(t, ConversationStatusActive, conversation.Status)
		assert.Empty(t, conversation.Messages)
		assert.NotZero(t, conversation.CreatedAt)
		assert.NotZero(t, conversation.UpdatedAt)

		// Verify no embedded foreign keys exist using reflection
		convType := reflect.TypeOf(*conversation)
		_, hasSessionID := convType.FieldByName("SessionID")
		_, hasUserID := convType.FieldByName("UserID")
		_, hasProjectID := convType.FieldByName("ProjectID")
		_, hasExecutionPlanIDs := convType.FieldByName("ExecutionPlanIDs")

		assert.False(t, hasSessionID, "Conversation should not have SessionID field")
		assert.False(t, hasUserID, "Conversation should not have UserID field")
		assert.False(t, hasProjectID, "Conversation should not have ProjectID field")
		assert.False(t, hasExecutionPlanIDs, "Conversation should not have ExecutionPlanIDs field")
	})

	t.Run("should fail with empty conversation ID", func(t *testing.T) {
		// When
		_, err := NewConversation("")

		// Then
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "conversation ID cannot be empty")
	})

	t.Run("should create conversation without requiring foreign key parameters", func(t *testing.T) {
		// This test ensures the graph-native principle - relationships via edges, not properties

		// When
		conversation, err := NewConversation("test-conv-id")

		// Then
		assert.NoError(t, err)
		assert.Equal(t, "test-conv-id", conversation.ID)
		assert.Equal(t, ConversationStatusActive, conversation.Status)

		// Relationships will be established via graph edges in repository layer
	})
}

func TestConversation_AddMessage(t *testing.T) {
	t.Run("should add user message", func(t *testing.T) {
		// Given
		conversation, _ := NewConversation("conv-123")
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
		conversation, _ := NewConversation("conv-123")
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
		conversation, _ := NewConversation("conv-123")
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
		conversation, _ := NewConversation("conv-123")

		// When
		err := conversation.AddMessage("", MessageRoleUser, "test", nil)

		// Then
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "message ID cannot be empty")
	})
}

func TestConversation_LinkExecutionPlan_GraphNative(t *testing.T) {
	t.Run("should handle execution plan linking in graph-native way", func(t *testing.T) {
		// Given
		conversation, _ := NewConversation("conv-123")
		planID := "plan-abc"

		// When
		err := conversation.LinkExecutionPlan(planID)

		// Then
		assert.NoError(t, err)
		// In graph-native architecture, execution plan relationships are handled by repository layer
		// This method only updates timestamp
	})

	t.Run("should fail with empty plan ID", func(t *testing.T) {
		// Given
		conversation, _ := NewConversation("conv-123")

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
		conversation, _ := NewConversation("conv-123")
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

	t.Run("should return empty slice for role with no messages", func(t *testing.T) {
		// Given
		conversation, _ := NewConversation("conv-123")

		// When
		systemMessages := conversation.GetMessagesByRole(MessageRoleSystem)

		// Then
		assert.Empty(t, systemMessages)
	})
}

func TestConversation_SetStatus(t *testing.T) {
	t.Run("should set conversation status", func(t *testing.T) {
		// Given
		conversation, _ := NewConversation("conv-123")
		initialUpdatedAt := conversation.UpdatedAt

		// When
		conversation.SetStatus(ConversationStatusPaused)

		// Then
		assert.Equal(t, ConversationStatusPaused, conversation.Status)
		assert.True(t, conversation.UpdatedAt.After(initialUpdatedAt))
	})
}

func TestConversation_Validate_GraphNative(t *testing.T) {
	t.Run("should validate graph-native conversation", func(t *testing.T) {
		// Given
		conversation, _ := NewConversation("conv-123")

		// When
		err := conversation.Validate()

		// Then
		assert.NoError(t, err)
	})

	t.Run("should fail validation with empty ID", func(t *testing.T) {
		// Given
		conversation := &Conversation{ID: ""}

		// When
		err := conversation.Validate()

		// Then
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "ID cannot be empty")
	})
}

func TestConversation_GetLatestMessage(t *testing.T) {
	t.Run("should return latest message by timestamp", func(t *testing.T) {
		// Given
		conversation, _ := NewConversation("conv-123")
		conversation.AddMessage("msg-1", MessageRoleUser, "First message", nil)
		conversation.AddMessage("msg-2", MessageRoleAssistant, "Second message", nil)
		conversation.AddMessage("msg-3", MessageRoleUser, "Latest message", nil)

		// When
		latestMessage := conversation.GetLatestMessage()

		// Then
		assert.NotNil(t, latestMessage)
		assert.Equal(t, "Latest message", latestMessage.Content)
		assert.Equal(t, "msg-3", latestMessage.ID)
	})

	t.Run("should return nil for conversation with no messages", func(t *testing.T) {
		// Given
		conversation, _ := NewConversation("conv-123")

		// When
		latestMessage := conversation.GetLatestMessage()

		// Then
		assert.Nil(t, latestMessage)
	})
}

func TestConversation_GetMessageCount(t *testing.T) {
	t.Run("should return correct message count", func(t *testing.T) {
		// Given
		conversation, _ := NewConversation("conv-123")
		conversation.AddMessage("msg-1", MessageRoleUser, "Message 1", nil)
		conversation.AddMessage("msg-2", MessageRoleAssistant, "Message 2", nil)

		// When
		count := conversation.GetMessageCount()

		// Then
		assert.Equal(t, 2, count)
	})

	t.Run("should return zero for conversation with no messages", func(t *testing.T) {
		// Given
		conversation, _ := NewConversation("conv-123")

		// When
		count := conversation.GetMessageCount()

		// Then
		assert.Equal(t, 0, count)
	})
}
