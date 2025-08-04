package domain

import (
	"fmt"
	"time"
)

// ConversationSummaryValidationError represents validation errors for conversation summaries
type ConversationSummaryValidationError struct {
	Field   string
	Message string
}

func (e ConversationSummaryValidationError) Error() string {
	return fmt.Sprintf("conversation summary validation error - %s: %s", e.Field, e.Message)
}

// SummaryType represents the type of conversation summary
type SummaryType string

const (
	SummaryTypeContextPreservation SummaryType = "context_preservation" // General conversation summary
	SummaryTypeClarificationChain  SummaryType = "clarification_chain"  // Focused on preserving clarification context
	SummaryTypeExecutionHistory    SummaryType = "execution_history"    // Focused on execution plan history
)

// ConversationSummary represents an AI-generated summary of conversation messages
// This enables intelligent context length management while preserving full conversation history in graph
type ConversationSummary struct {
	ID              string      `json:"id"`
	ConversationID  string      `json:"conversation_id"`
	Summary         string      `json:"summary"`          // AI-generated summary content
	TokenCount      int         `json:"token_count"`      // Estimated tokens in summary
	CoveredMessages []string    `json:"covered_messages"` // Message IDs summarized
	SummaryType     SummaryType `json:"summary_type"`     // Type of summary created
	CreatedAt       time.Time   `json:"created_at"`
	CreatedBy       string      `json:"created_by"` // AI provider that created summary
}

// NewConversationSummary creates a new conversation summary with validation
func NewConversationSummary(id, conversationID, summary string, coveredMessages []string, summaryType SummaryType) (*ConversationSummary, error) {
	if id == "" {
		return nil, ConversationSummaryValidationError{Field: "id", Message: "summary ID cannot be empty"}
	}

	if conversationID == "" {
		return nil, ConversationSummaryValidationError{Field: "conversation_id", Message: "conversation ID cannot be empty"}
	}

	if summary == "" {
		return nil, ConversationSummaryValidationError{Field: "summary", Message: "summary content cannot be empty"}
	}

	if len(coveredMessages) == 0 {
		return nil, ConversationSummaryValidationError{Field: "covered_messages", Message: "must include at least one message"}
	}

	// Validate summary type
	if summaryType != SummaryTypeContextPreservation &&
		summaryType != SummaryTypeClarificationChain &&
		summaryType != SummaryTypeExecutionHistory {
		return nil, ConversationSummaryValidationError{Field: "summary_type", Message: "invalid summary type"}
	}

	now := time.Now().UTC()

	return &ConversationSummary{
		ID:              id,
		ConversationID:  conversationID,
		Summary:         summary,
		TokenCount:      estimateTokenCount(summary),
		CoveredMessages: coveredMessages,
		SummaryType:     summaryType,
		CreatedAt:       now,
		CreatedBy:       "ai_summarizer", // Default to AI summarizer
	}, nil
}

// EstimateTokenCount estimates the number of tokens in the summary
// Simple approximation: 1 token ≈ 4 characters for OpenAI models
func estimateTokenCount(text string) int {
	return len(text) / 4
}

// UpdateTokenCount updates the token count estimate for the summary
func (cs *ConversationSummary) UpdateTokenCount() {
	cs.TokenCount = estimateTokenCount(cs.Summary)
}

// AddCoveredMessage adds a message ID to the covered messages list
func (cs *ConversationSummary) AddCoveredMessage(messageID string) error {
	if messageID == "" {
		return ConversationSummaryValidationError{Field: "message_id", Message: "message ID cannot be empty"}
	}

	// Check if message is already covered
	for _, id := range cs.CoveredMessages {
		if id == messageID {
			return nil // Already covered, no error
		}
	}

	cs.CoveredMessages = append(cs.CoveredMessages, messageID)
	return nil
}

// RemoveCoveredMessage removes a message ID from the covered messages list
func (cs *ConversationSummary) RemoveCoveredMessage(messageID string) {
	for i, id := range cs.CoveredMessages {
		if id == messageID {
			cs.CoveredMessages = append(cs.CoveredMessages[:i], cs.CoveredMessages[i+1:]...)
			break
		}
	}
}

// GetCoveredMessageCount returns the number of messages covered by this summary
func (cs *ConversationSummary) GetCoveredMessageCount() int {
	return len(cs.CoveredMessages)
}

// IsEmpty returns true if the summary contains no content
func (cs *ConversationSummary) IsEmpty() bool {
	return cs.Summary == "" || len(cs.CoveredMessages) == 0
}

// GetEfficiencyRatio returns the efficiency ratio of the summary
// (original message count / summary token count) - higher is better
func (cs *ConversationSummary) GetEfficiencyRatio() float64 {
	if cs.TokenCount == 0 {
		return 0
	}
	return float64(len(cs.CoveredMessages)) / float64(cs.TokenCount)
}
