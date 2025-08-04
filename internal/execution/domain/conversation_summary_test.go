package domain

import (
	"testing"
	"time"
)

func TestNewConversationSummary(t *testing.T) {
	conversationID := "conv-123"
	planID := "plan-456"
	summary := "This is the full conversation summary with technical details."
	userResult := "Simple answer for the user."

	cs, err := NewConversationSummary(conversationID, planID, summary, userResult)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if cs.ID == "" {
		t.Error("Expected ID to be generated")
	}

	if cs.ConversationID != conversationID {
		t.Errorf("Expected ConversationID %s, got %s", conversationID, cs.ConversationID)
	}

	if cs.PlanID != planID {
		t.Errorf("Expected PlanID %s, got %s", planID, cs.PlanID)
	}

	if cs.Summary != summary {
		t.Errorf("Expected Summary %s, got %s", summary, cs.Summary)
	}

	if cs.UserResult != userResult {
		t.Errorf("Expected UserResult %s, got %s", userResult, cs.UserResult)
	}

	if cs.Status != ConversationSummaryStatusCompleted {
		t.Errorf("Expected Status %s, got %s", ConversationSummaryStatusCompleted, cs.Status)
	}

	if cs.CreatedAt.IsZero() {
		t.Error("Expected CreatedAt to be set")
	}

	if cs.CompletedAt == nil {
		t.Error("Expected CompletedAt to be set")
	}
}

func TestNewConversationSummaryValidation(t *testing.T) {
	tests := []struct {
		name           string
		conversationID string
		planID         string
		summary        string
		userResult     string
		expectError    bool
	}{
		{
			name:           "valid input",
			conversationID: "conv-123",
			planID:         "plan-456",
			summary:        "Summary",
			userResult:     "Result",
			expectError:    false,
		},
		{
			name:           "empty conversation ID",
			conversationID: "",
			planID:         "plan-456",
			summary:        "Summary",
			userResult:     "Result",
			expectError:    true,
		},
		{
			name:           "empty plan ID",
			conversationID: "conv-123",
			planID:         "",
			summary:        "Summary",
			userResult:     "Result",
			expectError:    true,
		},
		{
			name:           "empty user result",
			conversationID: "conv-123",
			planID:         "plan-456",
			summary:        "Summary",
			userResult:     "",
			expectError:    true,
		},
		{
			name:           "empty summary is allowed",
			conversationID: "conv-123",
			planID:         "plan-456",
			summary:        "",
			userResult:     "Result",
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewConversationSummary(tt.conversationID, tt.planID, tt.summary, tt.userResult)
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestConversationSummaryFromContentExtraction(t *testing.T) {
	conversationID := "conv-123"
	planID := "plan-456"
	content := `{
  "user_answer": "This is the simple answer for the user",
  "conversation_summary": "This is a summary for the user"
}`

	cs, err := NewConversationSummaryFromContent(conversationID, planID, content)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	expectedUserResult := "This is the simple answer for the user"
	if cs.UserResult != expectedUserResult {
		t.Errorf("Expected UserResult %s, got %s", expectedUserResult, cs.UserResult)
	}

	if cs.Summary != content {
		t.Errorf("Expected Summary to be original content")
	}
}

func TestUpdateWithNewExecution(t *testing.T) {
	// Initial conversation summary
	cs, _ := NewConversationSummary("conv-123", "plan-456", "Initial summary", "Initial result")
	originalCreatedAt := cs.CreatedAt

	// Simulate some time passing
	time.Sleep(1 * time.Millisecond)

	// Update with new execution
	newContent := "Updated technical summary with more analysis"
	newUserResult := "Updated simple answer"

	err := cs.UpdateWithNewExecution("plan-789", newContent, newUserResult)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if cs.PlanID != "plan-789" {
		t.Errorf("Expected PlanID to be updated to plan-789, got %s", cs.PlanID)
	}

	if cs.Summary != newContent {
		t.Errorf("Expected Summary to be updated")
	}

	if cs.UserResult != newUserResult {
		t.Errorf("Expected UserResult to be updated")
	}

	if cs.CreatedAt != originalCreatedAt {
		t.Error("Expected CreatedAt to remain unchanged")
	}

	if cs.CompletedAt == nil {
		t.Error("Expected CompletedAt to be updated")
	}
}
