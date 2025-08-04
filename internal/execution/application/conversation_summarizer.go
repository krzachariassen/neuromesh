package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	aiDomain "neuromesh/internal/ai/domain"
	conversationApp "neuromesh/internal/conversation/application"
	executionDomain "neuromesh/internal/execution/domain"
	planningDomain "neuromesh/internal/planning/domain"
)

// ConversationSummarizer - AI-NATIVE CONVERSATION SUMMARIZATION
// This analyzes agent execution results and creates comprehensive conversation summaries
// with both technical depth and user-friendly content extraction
type ConversationSummarizer struct {
	aiProvider          aiDomain.AIProvider
	repository          planningDomain.ExecutionPlanRepository
	conversationService conversationApp.ConversationService
}

// NewConversationSummarizer creates a new AI-powered conversation summarizer
func NewConversationSummarizer(aiProvider aiDomain.AIProvider, repository planningDomain.ExecutionPlanRepository, conversationService conversationApp.ConversationService) *ConversationSummarizer {
	return &ConversationSummarizer{
		aiProvider:          aiProvider,
		repository:          repository,
		conversationService: conversationService,
	}
}

// SummarizeConversation creates a conversation summary from execution plan results
func (s *ConversationSummarizer) SummarizeConversation(ctx context.Context, conversationID, planID string) (*executionDomain.ConversationSummary, error) {
	// Validate dependencies (GREEN phase - add required nil checks)
	if s.aiProvider == nil {
		return nil, fmt.Errorf("aiProvider is required")
	}
	if s.repository == nil {
		return nil, fmt.Errorf("repository is required")
	}
	if s.conversationService == nil {
		return nil, fmt.Errorf("conversationService is required")
	}

	if conversationID == "" {
		return nil, fmt.Errorf("conversationID cannot be empty")
	}
	if planID == "" {
		return nil, fmt.Errorf("planID cannot be empty")
	}

	// Get conversation history for context-aware summarization
	conversationHistory, err := s.conversationService.GetConversationHistory(ctx, conversationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation history for %s: %w", conversationID, err)
	}

	// Get agent results directly for this execution plan
	agentResults, err := s.repository.GetAgentResultsByExecutionPlan(ctx, planID)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent results for plan %s: %w", planID, err)
	}

	// Validate we have some results to summarize
	if len(agentResults) == 0 {
		return nil, fmt.Errorf("no agent results found for execution plan %s", planID)
	}

	// Build conversation summary prompt with conversation history and agent results
	systemPrompt := s.BuildSummarizationSystemPrompt()
	userPrompt := s.BuildConversationAwareSummarizationPrompt(conversationHistory, agentResults)

	// Generate summary using AI with marker-based output
	summaryContent, err := s.aiProvider.CallAI(ctx, systemPrompt, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("AI summarization failed for plan %s: %w", planID, err)
	}

	if strings.TrimSpace(summaryContent) == "" {
		return nil, fmt.Errorf("AI summarization produced empty result for plan %s", planID)
	}

	// Create conversation summary from AI-generated content with marker extraction
	conversationSummary, err := executionDomain.NewConversationSummaryFromContent(conversationID, planID, summaryContent)
	if err != nil {
		return nil, fmt.Errorf("failed to create conversation summary: %w", err)
	}

	// Add summarization metadata
	successfulResults := make([]*executionDomain.AgentResult, 0)
	for _, result := range agentResults {
		if result.Status == executionDomain.AgentResultStatusSuccess {
			successfulResults = append(successfulResults, result)
		}
	}

	conversationSummary.Metadata["agent_count"] = len(agentResults)
	conversationSummary.Metadata["successful_agents"] = len(successfulResults)
	conversationSummary.Metadata["conversation_message_count"] = len(conversationHistory)
	conversationSummary.Metadata["summarization_timestamp"] = time.Now().Format("2006-01-02T15:04:05Z07:00")

	// Store the conversation summary in the graph database
	err = s.repository.StoreConversationSummary(ctx, conversationSummary)
	if err != nil {
		return nil, fmt.Errorf("failed to store conversation summary: %w", err)
	}

	return conversationSummary, nil
}

// BuildSummarizationSystemPrompt creates the system prompt for AI conversation summarization
func (s *ConversationSummarizer) BuildSummarizationSystemPrompt() string {
	return `You are an expert AI analyst specialized in creating comprehensive conversation summaries from multi-agent execution results.

CORE RESPONSIBILITIES:
1. Create conversation-aware summaries that capture the full user journey and interaction flow
2. Analyze agent execution results in the context of the entire conversation
3. Identify what the user originally wanted vs. what was actually executed
4. Create a unified, well-structured response that tells the story of the conversation
5. Handle mixed success/failure scenarios while maintaining conversation context

CONVERSATION SUMMARY FOCUS:
• Summarize the entire conversation flow, not just individual executions
• Reference the user's original intent and requests
• Explain how the conversation evolved and what was ultimately accomplished  
• Include context about any clarifications, limitations, or adjustments made
• Show the relationship between different parts of the conversation

CRITICAL OUTPUT FORMAT:
You MUST respond with a valid JSON object using this exact structure.
Do not include any text before or after the JSON:

{
  "user_answer": "Write a clear, simple answer that directly addresses what the user originally wanted to know. Reference the user's original request and what was accomplished. Be concise and written in plain language. Focus on the final outcome in context of the original intent. Make it clear what text/content was actually analyzed.",
  "conversation_summary": "Write a conversation summary that captures the full interaction. Describe what the user originally requested. Explain any clarifications or limitations that came up. Summarize what was ultimately accomplished. Note the relationship between the original request and final outcome."
}

EXAMPLE RESPONSE:
{
  "user_answer": "The text 'hello world' contains 2 words as requested.",
  "conversation_summary": "The user asked to count words in 'hello world'. The system successfully processed this request and determined there are 2 words in the phrase."
}

VALIDATION: Your response must be valid JSON that can be parsed. Test your JSON before responding.

EXAMPLE CONVERSATION FLOWS:
• User asks for multiple things, system clarifies capabilities, user adjusts request
• User provides text in one message, refers to it in later messages
• User makes ambiguous requests that require clarification

Your response should tell the complete story of this conversation, not just describe the final execution.`
}

// BuildConversationAwareSummarizationPrompt creates a user prompt that includes conversation history
func (s *ConversationSummarizer) BuildConversationAwareSummarizationPrompt(conversationHistory []*aiDomain.AIConversationMessage, agentResults []*executionDomain.AgentResult) string {
	var prompt strings.Builder

	prompt.WriteString("CONVERSATION-AWARE SUMMARY REQUEST\n\n")

	// Add conversation context first
	prompt.WriteString("CONVERSATION HISTORY:\n")
	for i, msg := range conversationHistory {
		prompt.WriteString(fmt.Sprintf("Message %d (%s): %s\n", i+1, msg.Role, msg.Content))
	}
	prompt.WriteString("\n")

	// Add execution results context
	successfulResults := make([]*executionDomain.AgentResult, 0)
	for _, result := range agentResults {
		if result.Status == executionDomain.AgentResultStatusSuccess {
			successfulResults = append(successfulResults, result)
		}
	}

	prompt.WriteString(fmt.Sprintf("Total Agent Results: %d\n", len(agentResults)))
	prompt.WriteString(fmt.Sprintf("Successful Results: %d\n", len(successfulResults)))
	prompt.WriteString(fmt.Sprintf("Failed Results: %d\n\n", len(agentResults)-len(successfulResults)))

	prompt.WriteString("AGENT EXECUTION RESULTS:\n\n")

	// Add each agent result with structured formatting
	for i, result := range agentResults {
		prompt.WriteString(fmt.Sprintf("=== AGENT RESULT %d ===\n", i+1))
		prompt.WriteString(fmt.Sprintf("Agent ID: %s\n", result.AgentID))
		prompt.WriteString(fmt.Sprintf("Execution Status: %s\n", result.Status))
		prompt.WriteString(fmt.Sprintf("Timestamp: %s\n", result.Timestamp.Format("2006-01-02 15:04:05")))
		prompt.WriteString(fmt.Sprintf("\nResult Content:\n%s\n", result.Content))

		// Add structured metadata
		if len(result.Metadata) > 0 {
			prompt.WriteString("\nResult Metadata:\n")
			for key, value := range result.Metadata {
				prompt.WriteString(fmt.Sprintf("  • %s: %v\n", key, value))
			}
		}
		prompt.WriteString("\n")
	}

	prompt.WriteString("CONVERSATION SUMMARY INSTRUCTIONS:\n")
	prompt.WriteString("Please create a conversation-aware summary that:\n")
	prompt.WriteString("1. Analyzes the full conversation flow from the conversation history\n")
	prompt.WriteString("2. Identifies the user's original intent and how it evolved\n")
	prompt.WriteString("3. Explains the relationship between different messages in the conversation\n")
	prompt.WriteString("4. Summarizes what was ultimately accomplished in context of the original request\n")
	prompt.WriteString("5. References specific content (like text to be analyzed) mentioned in earlier messages\n")
	prompt.WriteString("6. Creates a coherent narrative of the entire interaction\n\n")
	prompt.WriteString("Focus on the conversation as a whole, not just the final execution. Make sure the summary captures the user's journey and what they originally wanted to accomplish.")

	return prompt.String()
}
