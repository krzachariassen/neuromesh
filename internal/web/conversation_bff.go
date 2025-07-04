package web

import (
	"context"
	"fmt"
	"time"

	conversationApp "neuromesh/internal/conversation/application"
	conversationDomain "neuromesh/internal/conversation/domain"
	"neuromesh/internal/logging"
	orchestratorApp "neuromesh/internal/orchestrator/application"
	userApp "neuromesh/internal/user/application"
	userDomain "neuromesh/internal/user/domain"

	"github.com/google/uuid"
)

// ConversationAwareWebBFF extends WebBFF with conversation persistence capabilities
type ConversationAwareWebBFF struct {
	*WebBFF             // Embed existing WebBFF
	conversationService conversationApp.ConversationService
	userService         userApp.UserService
	logger              logging.Logger
}

// NewConversationAwareWebBFF creates a new conversation-aware WebBFF
func NewConversationAwareWebBFF(
	orchestrator AIOrchestrator,
	conversationService conversationApp.ConversationService,
	userService userApp.UserService,
	logger logging.Logger,
) *ConversationAwareWebBFF {
	webBFF := NewWebBFF(orchestrator, logger)

	return &ConversationAwareWebBFF{
		WebBFF:              webBFF,
		conversationService: conversationService,
		userService:         userService,
		logger:              logger,
	}
}

// ProcessWebMessageWithConversation processes a web message with full conversation persistence
func (w *ConversationAwareWebBFF) ProcessWebMessageWithConversation(ctx context.Context, sessionID, message string) (*WebResponse, error) {
	// Validate input
	if sessionID == "" {
		return nil, fmt.Errorf("session ID cannot be empty")
	}
	if message == "" {
		return nil, fmt.Errorf("message cannot be empty")
	}

	w.logger.Debug("Processing web message with conversation persistence",
		"sessionID", sessionID, "message", message)

	// 1. Ensure user and session exist
	user, _, err := w.ensureUserAndSession(ctx, sessionID)
	if err != nil {
		w.logger.Error("Failed to ensure user and session", err, "sessionID", sessionID)
		return w.handleError("Failed to initialize session", sessionID), nil
	}

	// 2. Get or create conversation for this session
	conversation, err := w.getOrCreateConversation(ctx, sessionID, user.ID)
	if err != nil {
		w.logger.Error("Failed to get or create conversation", err, "sessionID", sessionID)
		return w.handleError("Failed to initialize conversation", sessionID), nil
	}

	// 3. Add user message to conversation
	userMessageID := generateMessageID()
	err = w.conversationService.AddMessage(ctx, conversation.ID, userMessageID,
		conversationDomain.MessageRoleUser, message, nil)
	if err != nil {
		w.logger.Error("Failed to add user message to conversation", err,
			"conversationID", conversation.ID, "messageID", userMessageID)
		// Continue processing even if message storage fails
	}

	// 4. Process through orchestrator
	orchestratorRequest := &orchestratorApp.OrchestratorRequest{
		UserInput: message,
		UserID:    user.ID,
		SessionID: sessionID,
		MessageID: userMessageID, // Link orchestrator processing to the user message
	}

	aiResponse, err := w.processOrchestratorRequest(ctx, orchestratorRequest)
	if err != nil {
		w.logger.Error("Failed to process orchestrator request", err, "sessionID", sessionID)
		return w.handleError("Failed to process request", sessionID), nil
	}

	// 5. Add AI response to conversation
	assistantMessageID := generateMessageID()
	assistantMetadata := w.buildAssistantMetadata(aiResponse)

	err = w.conversationService.AddMessage(ctx, conversation.ID, assistantMessageID,
		conversationDomain.MessageRoleAssistant, aiResponse.Message, assistantMetadata)
	if err != nil {
		w.logger.Error("Failed to add assistant message to conversation", err,
			"conversationID", conversation.ID, "messageID", assistantMessageID)
		// Continue processing even if message storage fails
	}

	// 6. Link execution plan if created
	if aiResponse.ExecutionPlanID != "" {
		err = w.conversationService.LinkExecutionPlan(ctx, conversation.ID, aiResponse.ExecutionPlanID)
		if err != nil {
			w.logger.Error("Failed to link execution plan to conversation", err,
				"conversationID", conversation.ID, "executionPlanID", aiResponse.ExecutionPlanID)
			// Continue processing even if linking fails
		}
	}

	// 7. Build web response
	webResponse := w.buildWebResponse(aiResponse, sessionID)

	w.logger.Info("Web message processed with conversation persistence",
		"sessionID", sessionID, "conversationID", conversation.ID)

	return webResponse, nil
}

// ensureUserAndSession ensures that the user and session exist in the graph
func (w *ConversationAwareWebBFF) ensureUserAndSession(ctx context.Context, sessionID string) (*userDomain.User, *userDomain.Session, error) {
	// Check if user exists for this session
	userID := sessionID // Use sessionID as userID for web sessions

	user, err := w.userService.GetUser(ctx, userID)
	if err != nil {
		// User doesn't exist, create new user
		user, err = w.userService.CreateUser(ctx, userID, sessionID, userDomain.UserTypeWebSession)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create user: %w", err)
		}
		w.logger.Info("Created new user for web session", "userID", userID, "sessionID", sessionID)
	}

	// Check if session exists
	session, err := w.userService.GetSession(ctx, sessionID)
	if err != nil {
		// Session doesn't exist, create new session
		session, err = w.userService.CreateSession(ctx, sessionID, userID, 24*time.Hour)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create session: %w", err)
		}
		w.logger.Info("Created new session", "sessionID", sessionID, "userID", userID)
	}

	return user, session, nil
}

// getOrCreateConversation gets an existing conversation or creates a new one
func (w *ConversationAwareWebBFF) getOrCreateConversation(ctx context.Context, sessionID, userID string) (*conversationDomain.Conversation, error) {
	// Try to find active conversation for this session
	conversations, err := w.conversationService.FindConversationsBySession(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to find conversations for session: %w", err)
	}

	// Find active conversation
	for _, conv := range conversations {
		if conv.Status == conversationDomain.ConversationStatusActive {
			return conv, nil
		}
	}

	// No active conversation found, create new one
	conversationID := generateConversationID()
	conversation, err := w.conversationService.CreateConversation(ctx, conversationID, sessionID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to create conversation: %w", err)
	}

	w.logger.Info("Created new conversation", "conversationID", conversationID, "sessionID", sessionID)
	return conversation, nil
}

// processOrchestratorRequest processes the request through the orchestrator
func (w *ConversationAwareWebBFF) processOrchestratorRequest(ctx context.Context, request *orchestratorApp.OrchestratorRequest) (*orchestratorApp.OrchestratorResult, error) {
	// Use the existing orchestrator interface through the adapter pattern
	return w.orchestrator.ProcessRequest(ctx, request.UserInput, request.UserID)
}

// buildAssistantMetadata builds metadata for assistant messages
func (w *ConversationAwareWebBFF) buildAssistantMetadata(aiResponse *orchestratorApp.OrchestratorResult) map[string]interface{} {
	metadata := make(map[string]interface{})

	if aiResponse.Analysis != nil {
		metadata["analysis_intent"] = aiResponse.Analysis.Intent
		metadata["analysis_confidence"] = int64(aiResponse.Analysis.Confidence) // Ensure it's int64 for Neo4j

		// Handle required agents array - ensure it's not nil
		if aiResponse.Analysis.RequiredAgents != nil && len(aiResponse.Analysis.RequiredAgents) > 0 {
			metadata["required_agents"] = aiResponse.Analysis.RequiredAgents
		} else {
			// Don't store empty arrays in Neo4j
			metadata["required_agents"] = ""
		}
	}

	if aiResponse.Decision != nil {
		metadata["decision_type"] = string(aiResponse.Decision.Type)
		metadata["decision_reasoning"] = aiResponse.Decision.Reasoning
	}

	if aiResponse.ExecutionPlanID != "" {
		metadata["execution_plan_id"] = aiResponse.ExecutionPlanID
	}

	metadata["success"] = aiResponse.Success
	metadata["timestamp"] = time.Now().UTC().Format(time.RFC3339)

	return metadata
}

// buildWebResponse builds the web response from orchestrator result
func (w *ConversationAwareWebBFF) buildWebResponse(aiResponse *orchestratorApp.OrchestratorResult, sessionID string) *WebResponse {
	var intent string
	if aiResponse.Analysis != nil {
		intent = aiResponse.Analysis.Intent
	}

	webResponse := &WebResponse{
		Content:   aiResponse.Message,
		SessionID: sessionID,
		Intent:    intent,
	}

	if !aiResponse.Success {
		webResponse.Error = aiResponse.Error
	}

	return webResponse
}

// handleError creates an error response
func (w *ConversationAwareWebBFF) handleError(message, sessionID string) *WebResponse {
	return &WebResponse{
		Content:   "I'm sorry, I encountered an error processing your request.",
		SessionID: sessionID,
		Error:     message,
	}
}

// Utility functions for ID generation

// generateMessageID generates a unique message ID
func generateMessageID() string {
	return fmt.Sprintf("msg-%s", uuid.New().String())
}

// generateConversationID generates a unique conversation ID
func generateConversationID() string {
	return fmt.Sprintf("conv-%s", uuid.New().String())
}

// InitializeSchema ensures conversation and user schemas are created
func (w *ConversationAwareWebBFF) InitializeSchema(ctx context.Context) error {
	// Initialize user schema
	if err := w.userService.EnsureSchema(ctx); err != nil {
		return fmt.Errorf("failed to ensure user schema: %w", err)
	}

	// Initialize conversation schema
	if err := w.conversationService.EnsureSchema(ctx); err != nil {
		return fmt.Errorf("failed to ensure conversation schema: %w", err)
	}

	w.logger.Info("Conversation schemas initialized successfully")
	return nil
}
