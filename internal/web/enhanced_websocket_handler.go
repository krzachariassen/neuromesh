package web

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// EnhancedWebSocketConnection manages a single enhanced WebSocket connection
type EnhancedWebSocketConnection struct {
	conn      *websocket.Conn
	sessionID string
	bff       *ConversationAwareWebBFF
	done      chan struct{}
}

// NewEnhancedWebSocketConnection creates a new enhanced WebSocket connection
func NewEnhancedWebSocketConnection(conn *websocket.Conn, sessionID string, bff *ConversationAwareWebBFF) *EnhancedWebSocketConnection {
	return &EnhancedWebSocketConnection{
		conn:      conn,
		sessionID: sessionID,
		bff:       bff,
		done:      make(chan struct{}),
	}
}

// Start begins handling messages on this connection
func (ewsc *EnhancedWebSocketConnection) Start(ctx context.Context) {
	defer ewsc.conn.Close()
	defer close(ewsc.done)

	// Start a goroutine to handle periodic agent updates
	go ewsc.sendPeriodicAgentUpdates(ctx)

	// Handle incoming messages
	for {
		select {
		case <-ctx.Done():
			return
		case <-ewsc.done:
			return
		default:
			// Read message with timeout
			ewsc.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
			var rawMessage json.RawMessage
			if err := ewsc.conn.ReadJSON(&rawMessage); err != nil {
				ewsc.bff.logger.Error("Failed to read WebSocket message", err)
				ewsc.sendErrorMessage("invalid_message", "Failed to read invalid message", err.Error())
				return
			}

			// Process the message
			if err := ewsc.handleMessage(ctx, rawMessage); err != nil {
				ewsc.bff.logger.Error("Failed to handle WebSocket message", err)
				ewsc.sendErrorMessage("processing_error", "Failed to process message", err.Error())
			}
		}
	}
}

// handleMessage processes an incoming WebSocket message
func (ewsc *EnhancedWebSocketConnection) handleMessage(ctx context.Context, rawMessage json.RawMessage) error {
	var baseMessage struct {
		Type EnhancedWebSocketMessageType `json:"type"`
		ID   string                       `json:"id"`
	}

	if err := json.Unmarshal(rawMessage, &baseMessage); err != nil {
		return fmt.Errorf("failed to parse base message: %w", err)
	}

	switch baseMessage.Type {
	case EnhancedMessageTypeChatMessage:
		return ewsc.handleChatMessage(ctx, rawMessage)
	case EnhancedMessageTypePing:
		return ewsc.handlePingMessage(baseMessage.ID)
	default:
		return fmt.Errorf("unsupported message type: %s", baseMessage.Type)
	}
}

// handleChatMessage processes a chat message and triggers orchestration
func (ewsc *EnhancedWebSocketConnection) handleChatMessage(ctx context.Context, rawMessage json.RawMessage) error {
	var message EnhancedWebSocketMessage
	if err := json.Unmarshal(rawMessage, &message); err != nil {
		return fmt.Errorf("failed to parse chat message: %w", err)
	}

	// Extract chat data
	chatData, ok := message.Data.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid chat message data format")
	}

	content, ok := chatData["content"].(string)
	if !ok {
		return fmt.Errorf("missing or invalid content in chat message")
	}

	// Process message using existing BFF logic
	response, err := ewsc.bff.ProcessWebMessageWithConversation(ctx, ewsc.sessionID, content)
	if err != nil {
		return fmt.Errorf("failed to process message with conversation: %w", err)
	}

	// Send chat response FIRST
	responseMessage := EnhancedWebSocketMessage{
		Type:      EnhancedMessageTypeChatMessage,
		ID:        uuid.New().String(),
		Timestamp: time.Now(),
		Data: ChatMessageData{
			Content:        response.Content,
			Role:          "assistant",
			ConversationID: response.SessionID,
		},
	}

	if err := ewsc.conn.WriteJSON(responseMessage); err != nil {
		return fmt.Errorf("failed to send chat response: %w", err)
	}

	// Send execution start message (simulated for now)
	executionID := uuid.New().String()
	ewsc.sendExecutionStartMessage(executionID, content)

	// Send execution step message (simulated)
	ewsc.sendExecutionStepMessage(executionID, "analysis", "completed", "text-processor")

	return nil
}

// handlePingMessage responds to ping with pong
func (ewsc *EnhancedWebSocketConnection) handlePingMessage(id string) error {
	pongMessage := EnhancedWebSocketMessage{
		Type:      EnhancedMessageTypePong,
		ID:        id,
		Timestamp: time.Now(),
		SessionID: ewsc.sessionID,
		Data:      map[string]interface{}{"status": "ok"},
	}

	return ewsc.sendMessage(pongMessage)
}

// sendPeriodicAgentUpdates sends agent status updates periodically
func (ewsc *EnhancedWebSocketConnection) sendPeriodicAgentUpdates(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second) // Send updates every 10 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ewsc.done:
			return
		case <-ticker.C:
			// For MVP, send simulated agent updates
			// In production, this would come from actual agent registry
			agentUpdate := EnhancedWebSocketMessage{
				Type:      EnhancedMessageTypeAgentUpdate,
				ID:        uuid.New().String(),
				Timestamp: time.Now(),
				SessionID: ewsc.sessionID,
				Data: AgentUpdateData{
					AgentName: "text-processor",
					Type:      "processing",
					Status:    "active",
					Capabilities: []string{"text_analysis", "nlp_processing"},
					Metadata: struct {
						LastActive string `json:"last_active"`
					}{
						LastActive: time.Now().Format(time.RFC3339),
					},
				},
			}

			if err := ewsc.sendMessage(agentUpdate); err != nil {
				ewsc.bff.logger.Error("Failed to send agent update", err)
				// Don't break the connection for periodic update failures
			}
		}
	}
}

// sendExecutionStartMessage sends execution start notification
func (ewsc *EnhancedWebSocketConnection) sendExecutionStartMessage(executionID, content string) {
	message := EnhancedWebSocketMessage{
		Type:      EnhancedMessageTypeExecutionStart,
		ID:        uuid.New().String(),
		Timestamp: time.Now(),
		SessionID: ewsc.sessionID,
		Data: ExecutionStartData{
			ExecutionID:    executionID,
			ConversationID: ewsc.sessionID, // Using session as conversation for MVP
			PlanID:         uuid.New().String(),
			StartTime:      time.Now(),
			EstimatedSteps: 2, // Simple estimation
		},
	}

	if err := ewsc.sendMessage(message); err != nil {
		ewsc.bff.logger.Error("Failed to send execution start message", err)
	}
}

// sendExecutionStepMessage sends execution step update
func (ewsc *EnhancedWebSocketConnection) sendExecutionStepMessage(executionID, stepName, status, agentName string) {
	message := EnhancedWebSocketMessage{
		Type:      EnhancedMessageTypeExecutionStep,
		ID:        uuid.New().String(),
		Timestamp: time.Now(),
		SessionID: ewsc.sessionID,
		Data: EnhancedExecutionStepData{
			ExecutionStepData: ExecutionStepData{
				StepNumber:  1,
				Name:        stepName,
				Description: "Processing user request",
				AgentName:   agentName,
				Status:      status,
			},
			ExecutionID: executionID,
			StartTime:   time.Now(),
			Result:      "Step completed successfully",
		},
	}

	endTime := time.Now()
	if status == "completed" {
		if data, ok := message.Data.(EnhancedExecutionStepData); ok {
			data.EndTime = &endTime
			message.Data = data
		}
	}

	if err := ewsc.sendMessage(message); err != nil {
		ewsc.bff.logger.Error("Failed to send execution step message", err)
	}
}

// sendErrorMessage sends a structured error message
func (ewsc *EnhancedWebSocketConnection) sendErrorMessage(code, message, details string) {
	errorMessage := EnhancedWebSocketMessage{
		Type:      EnhancedMessageTypeError,
		ID:        uuid.New().String(),
		Timestamp: time.Now(),
		SessionID: ewsc.sessionID,
		Error: &ErrorData{
			Code:    code,
			Message: message,
			Details: details,
		},
	}

	if err := ewsc.sendMessage(errorMessage); err != nil {
		ewsc.bff.logger.Error("Failed to send error message", err)
	}
}

// sendMessage sends a message to the WebSocket connection
func (ewsc *EnhancedWebSocketConnection) sendMessage(message EnhancedWebSocketMessage) error {
	ewsc.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	return ewsc.conn.WriteJSON(message)
}
