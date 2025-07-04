package infrastructure

import (
	"context"
	"fmt"
	"time"

	"neuromesh/internal/conversation/domain"
	"neuromesh/internal/graph"
)

// Constants for graph node types and relationships
const (
	NodeTypeConversation = "Conversation"
	NodeTypeMessage      = "ConversationMessage"

	RelationshipBelongsToConversation = "BELONGS_TO_CONVERSATION"
	RelationshipContainsMessage       = "CONTAINS_MESSAGE"
	RelationshipInSession             = "IN_SESSION"
	RelationshipParticipantIn         = "PARTICIPANT_IN"
	RelationshipLinkedToPlan          = "LINKED_TO_PLAN"

	TimeFormat = "2006-01-02T15:04:05Z"
)

// GraphConversationRepository implements conversation repository using the graph backend
type GraphConversationRepository struct {
	graph graph.Graph
}

// NewGraphConversationRepository creates a new graph-based conversation repository
func NewGraphConversationRepository(g graph.Graph) domain.ConversationRepository {
	return &GraphConversationRepository{
		graph: g,
	}
}

// formatTime formats time for graph storage
func formatTime(t time.Time) string {
	return t.Format(TimeFormat)
}

// parseTime parses time from graph storage
func parseTime(timeStr string) (time.Time, error) {
	return time.Parse(TimeFormat, timeStr)
}

// EnsureConversationSchema ensures that the required schema for Conversation domain is in place
func (r *GraphConversationRepository) EnsureConversationSchema(ctx context.Context) error {
	// Create unique constraints for Conversation nodes
	if err := r.graph.CreateUniqueConstraint(ctx, NodeTypeConversation, "id"); err != nil {
		return fmt.Errorf("failed to create conversation id constraint: %w", err)
	}

	// Create indexes for Conversation nodes
	conversationIndexes := []string{"user_id", "session_id", "status", "created_at", "updated_at"}
	for _, property := range conversationIndexes {
		if err := r.graph.CreateIndex(ctx, NodeTypeConversation, property); err != nil {
			return fmt.Errorf("failed to create conversation %s index: %w", property, err)
		}
	}

	return nil
}

// EnsureMessageSchema ensures that the required schema for Message domain is in place
func (r *GraphConversationRepository) EnsureMessageSchema(ctx context.Context) error {
	// Create unique constraints for Message nodes
	if err := r.graph.CreateUniqueConstraint(ctx, NodeTypeMessage, "id"); err != nil {
		return fmt.Errorf("failed to create message id constraint: %w", err)
	}

	// Create indexes for Message nodes
	messageIndexes := []string{"conversation_id", "role", "timestamp"}
	for _, property := range messageIndexes {
		if err := r.graph.CreateIndex(ctx, NodeTypeMessage, property); err != nil {
			return fmt.Errorf("failed to create message %s index: %w", property, err)
		}
	}

	return nil
}

// CreateConversation creates a conversation node in the graph
func (r *GraphConversationRepository) CreateConversation(ctx context.Context, conversation *domain.Conversation) error {
	properties := map[string]interface{}{
		"id":                 conversation.ID,
		"session_id":         conversation.SessionID,
		"user_id":            conversation.UserID,
		"status":             string(conversation.Status),
		"execution_plan_ids": conversation.ExecutionPlanIDs,
		"created_at":         formatTime(conversation.CreatedAt),
		"updated_at":         formatTime(conversation.UpdatedAt),
	}

	return r.graph.AddNode(ctx, NodeTypeConversation, conversation.ID, properties)
}

// GetConversation retrieves a conversation by ID
func (r *GraphConversationRepository) GetConversation(ctx context.Context, conversationID string) (*domain.Conversation, error) {
	conversationProps, err := r.graph.GetNode(ctx, NodeTypeConversation, conversationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation: %w", err)
	}

	if conversationProps == nil {
		return nil, fmt.Errorf("conversation not found: %s", conversationID)
	}

	return r.mapToConversation(conversationProps)
}

// GetConversationWithMessages retrieves a conversation with all its messages
func (r *GraphConversationRepository) GetConversationWithMessages(ctx context.Context, conversationID string) (*domain.Conversation, error) {
	// Get the conversation
	conversation, err := r.GetConversation(ctx, conversationID)
	if err != nil {
		return nil, err
	}

	// Get the messages
	messages, err := r.GetConversationMessages(ctx, conversationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation messages: %w", err)
	}

	conversation.Messages = messages
	return conversation, nil
}

// UpdateConversation updates a conversation node in the graph
func (r *GraphConversationRepository) UpdateConversation(ctx context.Context, conversation *domain.Conversation) error {
	properties := map[string]interface{}{
		"session_id":         conversation.SessionID,
		"user_id":            conversation.UserID,
		"status":             string(conversation.Status),
		"execution_plan_ids": conversation.ExecutionPlanIDs,
		"updated_at":         formatTime(conversation.UpdatedAt),
	}

	return r.graph.UpdateNode(ctx, NodeTypeConversation, conversation.ID, properties)
}

// DeleteConversation deletes a conversation node from the graph
func (r *GraphConversationRepository) DeleteConversation(ctx context.Context, conversationID string) error {
	return r.graph.DeleteNode(ctx, NodeTypeConversation, conversationID)
}

// AddMessage adds a message to a conversation
func (r *GraphConversationRepository) AddMessage(ctx context.Context, conversationID string, message *domain.ConversationMessage) error {
	// Create message node
	properties := map[string]interface{}{
		"id":              message.ID,
		"conversation_id": conversationID,
		"role":            string(message.Role),
		"content":         message.Content,
		"timestamp":       formatTime(message.Timestamp),
	}

	// Only add metadata if it's not nil and not empty
	if message.Metadata != nil && len(message.Metadata) > 0 {
		properties["metadata"] = message.Metadata
	}

	if err := r.graph.AddNode(ctx, NodeTypeMessage, message.ID, properties); err != nil {
		return fmt.Errorf("failed to create message node: %w", err)
	}

	// Create relationship between conversation and message
	relationshipProps := map[string]interface{}{
		"created_at": formatTime(time.Now().UTC()),
	}

	return r.graph.AddEdge(ctx, NodeTypeConversation, conversationID, NodeTypeMessage, message.ID, RelationshipContainsMessage, relationshipProps)
}

// GetConversationMessages retrieves all messages for a conversation
func (r *GraphConversationRepository) GetConversationMessages(ctx context.Context, conversationID string) ([]domain.ConversationMessage, error) {
	// Query messages by conversation_id
	filters := map[string]interface{}{
		"conversation_id": conversationID,
	}

	messageProps, err := r.graph.QueryNodes(ctx, NodeTypeMessage, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to query conversation messages: %w", err)
	}

	messages := make([]domain.ConversationMessage, len(messageProps))
	for i, props := range messageProps {
		message, err := r.mapToMessage(props)
		if err != nil {
			return nil, fmt.Errorf("failed to map message properties: %w", err)
		}
		messages[i] = *message
	}

	return messages, nil
}

// GetMessagesByRole retrieves messages by role for a conversation
func (r *GraphConversationRepository) GetMessagesByRole(ctx context.Context, conversationID string, role domain.MessageRole) ([]domain.ConversationMessage, error) {
	// Query messages by conversation_id and role
	filters := map[string]interface{}{
		"conversation_id": conversationID,
		"role":            string(role),
	}

	messageProps, err := r.graph.QueryNodes(ctx, NodeTypeMessage, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to query messages by role: %w", err)
	}

	messages := make([]domain.ConversationMessage, len(messageProps))
	for i, props := range messageProps {
		message, err := r.mapToMessage(props)
		if err != nil {
			return nil, fmt.Errorf("failed to map message properties: %w", err)
		}
		messages[i] = *message
	}

	return messages, nil
}

// LinkConversationToSession creates a relationship between conversation and session
func (r *GraphConversationRepository) LinkConversationToSession(ctx context.Context, conversationID, sessionID string) error {
	properties := map[string]interface{}{
		"created_at": formatTime(time.Now().UTC()),
	}

	return r.graph.AddEdge(ctx, "Session", sessionID, NodeTypeConversation, conversationID, RelationshipInSession, properties)
}

// LinkConversationToUser creates a relationship between conversation and user
func (r *GraphConversationRepository) LinkConversationToUser(ctx context.Context, conversationID, userID string) error {
	properties := map[string]interface{}{
		"created_at": formatTime(time.Now().UTC()),
	}

	return r.graph.AddEdge(ctx, "User", userID, NodeTypeConversation, conversationID, RelationshipParticipantIn, properties)
}

// LinkExecutionPlan creates a relationship between conversation and execution plan
func (r *GraphConversationRepository) LinkExecutionPlan(ctx context.Context, conversationID, planID string) error {
	properties := map[string]interface{}{
		"created_at": formatTime(time.Now().UTC()),
	}

	return r.graph.AddEdge(ctx, NodeTypeConversation, conversationID, "ExecutionPlan", planID, RelationshipLinkedToPlan, properties)
}

// FindConversationsByUser finds conversations by user ID
func (r *GraphConversationRepository) FindConversationsByUser(ctx context.Context, userID string) ([]*domain.Conversation, error) {
	filters := map[string]interface{}{
		"user_id": userID,
	}

	conversationProps, err := r.graph.QueryNodes(ctx, NodeTypeConversation, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to query conversations by user: %w", err)
	}

	conversations := make([]*domain.Conversation, len(conversationProps))
	for i, props := range conversationProps {
		conversation, err := r.mapToConversation(props)
		if err != nil {
			return nil, fmt.Errorf("failed to map conversation properties: %w", err)
		}
		conversations[i] = conversation
	}

	return conversations, nil
}

// FindConversationsBySession finds conversations by session ID
func (r *GraphConversationRepository) FindConversationsBySession(ctx context.Context, sessionID string) ([]*domain.Conversation, error) {
	filters := map[string]interface{}{
		"session_id": sessionID,
	}

	conversationProps, err := r.graph.QueryNodes(ctx, NodeTypeConversation, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to query conversations by session: %w", err)
	}

	conversations := make([]*domain.Conversation, len(conversationProps))
	for i, props := range conversationProps {
		conversation, err := r.mapToConversation(props)
		if err != nil {
			return nil, fmt.Errorf("failed to map conversation properties: %w", err)
		}
		conversations[i] = conversation
	}

	return conversations, nil
}

// FindActiveConversations finds all active conversations
func (r *GraphConversationRepository) FindActiveConversations(ctx context.Context) ([]*domain.Conversation, error) {
	return r.FindConversationsByStatus(ctx, domain.ConversationStatusActive)
}

// FindConversationsByStatus finds conversations by status
func (r *GraphConversationRepository) FindConversationsByStatus(ctx context.Context, status domain.ConversationStatus) ([]*domain.Conversation, error) {
	filters := map[string]interface{}{
		"status": string(status),
	}

	conversationProps, err := r.graph.QueryNodes(ctx, NodeTypeConversation, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to query conversations by status: %w", err)
	}

	conversations := make([]*domain.Conversation, len(conversationProps))
	for i, props := range conversationProps {
		conversation, err := r.mapToConversation(props)
		if err != nil {
			return nil, fmt.Errorf("failed to map conversation properties: %w", err)
		}
		conversations[i] = conversation
	}

	return conversations, nil
}

// mapToConversation converts map properties to Conversation domain object
func (r *GraphConversationRepository) mapToConversation(props map[string]interface{}) (*domain.Conversation, error) {
	id, ok := props["id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid conversation id")
	}

	sessionID, ok := props["session_id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid session_id")
	}

	userID, ok := props["user_id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid user_id")
	}

	statusStr, ok := props["status"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid status")
	}

	createdAtStr, ok := props["created_at"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid created_at")
	}

	updatedAtStr, ok := props["updated_at"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid updated_at")
	}

	// Parse timestamps
	createdAt, err := parseTime(createdAtStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse created_at: %w", err)
	}

	updatedAt, err := parseTime(updatedAtStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse updated_at: %w", err)
	}

	// Handle execution plan IDs (may be nil or array)
	var executionPlanIDs []string
	if planIDs, exists := props["execution_plan_ids"]; exists && planIDs != nil {
		if planIDsSlice, ok := planIDs.([]interface{}); ok {
			executionPlanIDs = make([]string, len(planIDsSlice))
			for i, planID := range planIDsSlice {
				if planIDStr, ok := planID.(string); ok {
					executionPlanIDs[i] = planIDStr
				}
			}
		}
	}

	if executionPlanIDs == nil {
		executionPlanIDs = make([]string, 0)
	}

	// Create conversation object
	conversation := &domain.Conversation{
		ID:               id,
		SessionID:        sessionID,
		UserID:           userID,
		Status:           domain.ConversationStatus(statusStr),
		Messages:         make([]domain.ConversationMessage, 0), // Messages loaded separately
		ExecutionPlanIDs: executionPlanIDs,
		CreatedAt:        createdAt,
		UpdatedAt:        updatedAt,
	}

	return conversation, nil
}

// mapToMessage converts map properties to ConversationMessage domain object
func (r *GraphConversationRepository) mapToMessage(props map[string]interface{}) (*domain.ConversationMessage, error) {
	id, ok := props["id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid message id")
	}

	roleStr, ok := props["role"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid role")
	}

	content, ok := props["content"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid content")
	}

	timestampStr, ok := props["timestamp"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid timestamp")
	}

	// Parse timestamp
	timestamp, err := parseTime(timestampStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse timestamp: %w", err)
	}

	// Handle metadata (may be nil)
	metadata := make(map[string]interface{})
	if metadataRaw, exists := props["metadata"]; exists && metadataRaw != nil {
		if metadataMap, ok := metadataRaw.(map[string]interface{}); ok {
			metadata = metadataMap
		}
	}

	// Create message object
	message := &domain.ConversationMessage{
		ID:        id,
		Role:      domain.MessageRole(roleStr),
		Content:   content,
		Timestamp: timestamp,
		Metadata:  metadata,
	}

	return message, nil
}
