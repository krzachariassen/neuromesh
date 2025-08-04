package domain

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ConversationSummaryStatus represents the status of conversation summary
type ConversationSummaryStatus string

const (
	ConversationSummaryStatusPending   ConversationSummaryStatus = "PENDING"
	ConversationSummaryStatusCompleted ConversationSummaryStatus = "COMPLETED"
	ConversationSummaryStatusFailed    ConversationSummaryStatus = "FAILED"
)

// ConversationSummary represents the summary of a conversation including both technical details and user-friendly results
type ConversationSummary struct {
	ID             string                    `json:"id"`
	ConversationID string                    `json:"conversation_id"`
	PlanID         string                    `json:"plan_id"`     // Link to the execution plan that generated this summary
	Summary        string                    `json:"summary"`     // Full technical conversation summary (was Content)
	UserResult     string                    `json:"user_result"` // Simple user-friendly answer (was UserAnswer)
	Status         ConversationSummaryStatus `json:"status"`
	CreatedAt      time.Time                 `json:"created_at"`
	CompletedAt    *time.Time                `json:"completed_at,omitempty"`
	ErrorMessage   string                    `json:"error_message,omitempty"`
	Metadata       map[string]interface{}    `json:"metadata,omitempty"`
}

// NewConversationSummary creates a new conversation summary
func NewConversationSummary(conversationID, planID, summary, userResult string) (*ConversationSummary, error) {
	if conversationID == "" {
		return nil, fmt.Errorf("conversation ID cannot be empty")
	}
	if planID == "" {
		return nil, fmt.Errorf("plan ID cannot be empty")
	}
	if userResult == "" {
		return nil, fmt.Errorf("user result cannot be empty")
	}

	return &ConversationSummary{
		ID:             uuid.New().String(),
		ConversationID: conversationID,
		PlanID:         planID,
		Summary:        summary,
		UserResult:     userResult,
		Status:         ConversationSummaryStatusCompleted,
		CreatedAt:      time.Now(),
		CompletedAt:    func() *time.Time { t := time.Now(); return &t }(),
		Metadata:       make(map[string]interface{}),
	}, nil
}

// NewPendingConversationSummary creates a conversation summary in pending state
func NewPendingConversationSummary(conversationID, planID string) (*ConversationSummary, error) {
	if conversationID == "" {
		return nil, fmt.Errorf("conversation ID cannot be empty")
	}
	if planID == "" {
		return nil, fmt.Errorf("plan ID cannot be empty")
	}

	return &ConversationSummary{
		ID:             uuid.New().String(),
		ConversationID: conversationID,
		PlanID:         planID,
		Status:         ConversationSummaryStatusPending,
		CreatedAt:      time.Now(),
		Metadata:       make(map[string]interface{}),
	}, nil
}

// NewConversationSummaryFromContent creates a conversation summary by extracting user-friendly content
func NewConversationSummaryFromContent(conversationID, planID, content string) (*ConversationSummary, error) {
	userResult, err := ExtractUserFriendlyContent(content)
	if err != nil {
		return nil, fmt.Errorf("failed to extract user result: %w", err)
	}

	return NewConversationSummary(conversationID, planID, content, userResult.Answer)
}

// UpdateWithNewExecution updates the conversation summary with results from a new execution
func (cs *ConversationSummary) UpdateWithNewExecution(newPlanID, newSummary, newUserResult string) error {
	if newPlanID == "" {
		return fmt.Errorf("new plan ID cannot be empty")
	}
	if newUserResult == "" {
		return fmt.Errorf("new user result cannot be empty")
	}

	cs.PlanID = newPlanID
	cs.Summary = newSummary
	cs.UserResult = newUserResult
	cs.Status = ConversationSummaryStatusCompleted
	now := time.Now()
	cs.CompletedAt = &now

	return nil
}

// MarkFailed marks the conversation summary as failed
func (cs *ConversationSummary) MarkFailed(errorMessage string) {
	cs.Status = ConversationSummaryStatusFailed
	cs.ErrorMessage = errorMessage
	now := time.Now()
	cs.CompletedAt = &now
}

// MarkCompleted marks the conversation summary as completed
func (cs *ConversationSummary) MarkCompleted() {
	cs.Status = ConversationSummaryStatusCompleted
	now := time.Now()
	cs.CompletedAt = &now
}

// Validate validates the conversation summary
func (cs *ConversationSummary) Validate() error {
	if cs.ID == "" {
		return fmt.Errorf("conversation summary ID cannot be empty")
	}
	if cs.ConversationID == "" {
		return fmt.Errorf("conversation ID cannot be empty")
	}
	if cs.PlanID == "" {
		return fmt.Errorf("plan ID cannot be empty")
	}
	if cs.Status == ConversationSummaryStatusCompleted && cs.UserResult == "" {
		return fmt.Errorf("user result cannot be empty for completed summary")
	}
	return nil
}

// IsCompleted returns true if the conversation summary is completed
func (cs *ConversationSummary) IsCompleted() bool {
	return cs.Status == ConversationSummaryStatusCompleted
}

// IsFailed returns true if the conversation summary failed
func (cs *ConversationSummary) IsFailed() bool {
	return cs.Status == ConversationSummaryStatusFailed
}

// IsPending returns true if the conversation summary is pending
func (cs *ConversationSummary) IsPending() bool {
	return cs.Status == ConversationSummaryStatusPending
}

// UserFriendlyContent represents extracted user-friendly information from AI-generated content
type UserFriendlyContent struct {
	Answer  string `json:"answer"`
	Summary string `json:"summary"`
}

// NewUserFriendlyContent creates a new user-friendly content value object
func NewUserFriendlyContent(answer, summary string) *UserFriendlyContent {
	return &UserFriendlyContent{
		Answer:  strings.TrimSpace(answer),
		Summary: strings.TrimSpace(summary),
	}
}

// Validate ensures the user-friendly content is valid
func (ufc *UserFriendlyContent) Validate() error {
	if ufc.Answer == "" {
		return fmt.Errorf("user answer cannot be empty")
	}
	if ufc.Summary == "" {
		return fmt.Errorf("user summary cannot be empty")
	}
	return nil
}

// ExtractUserFriendlyContent extracts user-friendly content from AI-generated JSON
// The AI must respond with a JSON object containing user_answer and conversation_summary fields
func ExtractUserFriendlyContent(content string) (*UserFriendlyContent, error) {
	if content == "" {
		return nil, fmt.Errorf("content cannot be empty")
	}

	// Parse JSON response from AI
	var aiResponse struct {
		UserAnswer          string `json:"user_answer"`
		ConversationSummary string `json:"conversation_summary"`
	}

	if err := json.Unmarshal([]byte(strings.TrimSpace(content)), &aiResponse); err != nil {
		return nil, fmt.Errorf("failed to parse AI JSON response: %w", err)
	}

	// Create value object and validate
	userContent := NewUserFriendlyContent(
		strings.TrimSpace(aiResponse.UserAnswer),
		strings.TrimSpace(aiResponse.ConversationSummary),
	)
	if err := userContent.Validate(); err != nil {
		return nil, fmt.Errorf("invalid user-friendly content: %w", err)
	}

	return userContent, nil
}
