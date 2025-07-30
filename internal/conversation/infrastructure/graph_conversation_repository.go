package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"neuromesh/internal/conversation/domain"
	"neuromesh/internal/graph"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
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
	RelationshipBelongsTo             = "BELONGS_TO"

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
	conversationIndexes := []string{"user_id", "session_id", "project_id", "status", "created_at", "updated_at"}
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

// CreateConversation creates a conversation node in the graph without embedded foreign keys
func (r *GraphConversationRepository) CreateConversation(ctx context.Context, conversation *domain.Conversation) error {
	// Graph-native: Only store essential conversation properties, relationships are handled separately
	properties := map[string]interface{}{
		"id":         conversation.ID,
		"status":     string(conversation.Status),
		"created_at": formatTime(conversation.CreatedAt),
		"updated_at": formatTime(conversation.UpdatedAt),
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

// UpdateConversation updates a conversation node in the graph (graph-native approach)
func (r *GraphConversationRepository) UpdateConversation(ctx context.Context, conversation *domain.Conversation) error {
	// Graph-native: Only update essential conversation properties, relationships are managed separately
	properties := map[string]interface{}{
		"status":     string(conversation.Status),
		"updated_at": formatTime(conversation.UpdatedAt),
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

	// Only add metadata if it's not empty
	if len(message.Metadata) > 0 {
		// Neo4j can't handle nested maps, so serialize to JSON string
		metadataJSON, err := json.Marshal(message.Metadata)
		if err != nil {
			return fmt.Errorf("failed to serialize metadata: %w", err)
		}
		properties["metadata"] = string(metadataJSON)
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

// LinkConversationToProject creates a relationship between conversation and project
func (r *GraphConversationRepository) LinkConversationToProject(ctx context.Context, conversationID, projectID string) error {
	properties := map[string]interface{}{
		"created_at": formatTime(time.Now().UTC()),
	}

	// Link conversation to project using BELONGS_TO relationship
	// CRITICAL: Use "Project" (capital P) to match the project domain node type
	return r.graph.AddEdge(ctx, NodeTypeConversation, conversationID, "Project", projectID, "BELONGS_TO", properties)
}

// LinkExecutionPlan creates a relationship between conversation and execution plan
func (r *GraphConversationRepository) LinkExecutionPlan(ctx context.Context, conversationID, planID string) error {
	properties := map[string]interface{}{
		"created_at": formatTime(time.Now().UTC()),
	}

	// Use the correct node type for execution plan - must match the planning domain
	return r.graph.AddEdge(ctx, NodeTypeConversation, conversationID, "execution_plan", planID, RelationshipLinkedToPlan, properties)
}

// FindConversationsByUser finds conversations by user ID using graph relationships (graph-native)
func (r *GraphConversationRepository) FindConversationsByUser(ctx context.Context, userID string) ([]*domain.Conversation, error) {
	// Type assert to get Neo4j driver access for direct Cypher queries
	neo4jGraph, ok := r.graph.(*graph.Neo4jGraph)
	if !ok {
		return nil, fmt.Errorf("FindConversationsByUser requires Neo4j graph implementation")
	}

	driver := neo4jGraph.Driver()
	session := driver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	// Cypher query to traverse relationships and find conversations for user
	query := `
		MATCH (u:User {id: $userID})-[:PARTICIPANT_IN]->(c:Conversation)
		RETURN c.id as id, c.status as status, c.created_at as created_at, c.updated_at as updated_at
		ORDER BY c.created_at DESC
	`

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		result, err := tx.Run(ctx, query, map[string]interface{}{
			"userID": userID,
		})
		if err != nil {
			return nil, err
		}

		var conversations []*domain.Conversation
		for result.Next(ctx) {
			record := result.Record()

			// Extract conversation properties
			id := record.Values[0].(string)
			status := record.Values[1].(string)
			createdAtStr := record.Values[2].(string)
			updatedAtStr := record.Values[3].(string)

			// Parse timestamps
			createdAt, err := parseTime(createdAtStr)
			if err != nil {
				return nil, fmt.Errorf("failed to parse created_at: %w", err)
			}

			updatedAt, err := parseTime(updatedAtStr)
			if err != nil {
				return nil, fmt.Errorf("failed to parse updated_at: %w", err)
			}

			// Create conversation object
			conversation := &domain.Conversation{
				ID:        id,
				Status:    domain.ConversationStatus(status),
				Messages:  make([]domain.ConversationMessage, 0),
				CreatedAt: createdAt,
				UpdatedAt: updatedAt,
			}

			conversations = append(conversations, conversation)
		}

		return conversations, result.Err()
	})

	if err != nil {
		return nil, fmt.Errorf("failed to find conversations by user: %w", err)
	}

	return result.([]*domain.Conversation), nil
} // FindConversationsBySession finds conversations by session ID using graph relationships (graph-native)
func (r *GraphConversationRepository) FindConversationsBySession(ctx context.Context, sessionID string) ([]*domain.Conversation, error) {
	// Type assert to get Neo4j driver access for direct Cypher queries
	neo4jGraph, ok := r.graph.(*graph.Neo4jGraph)
	if !ok {
		return nil, fmt.Errorf("FindConversationsBySession requires Neo4j graph implementation")
	}

	driver := neo4jGraph.Driver()
	session := driver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	// Cypher query to traverse relationships and find conversations for session
	query := `
		MATCH (s:Session {id: $sessionID})-[:IN_SESSION]-(c:Conversation)
		RETURN c.id as id, c.status as status, c.created_at as created_at, c.updated_at as updated_at
		ORDER BY c.created_at DESC
	`

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		result, err := tx.Run(ctx, query, map[string]interface{}{
			"sessionID": sessionID,
		})
		if err != nil {
			return nil, err
		}

		var conversations []*domain.Conversation
		for result.Next(ctx) {
			record := result.Record()

			// Extract conversation properties
			id := record.Values[0].(string)
			status := record.Values[1].(string)
			createdAtStr := record.Values[2].(string)
			updatedAtStr := record.Values[3].(string)

			// Parse timestamps
			createdAt, err := parseTime(createdAtStr)
			if err != nil {
				return nil, fmt.Errorf("failed to parse created_at: %w", err)
			}

			updatedAt, err := parseTime(updatedAtStr)
			if err != nil {
				return nil, fmt.Errorf("failed to parse updated_at: %w", err)
			}

			// Create conversation object
			conversation := &domain.Conversation{
				ID:        id,
				Status:    domain.ConversationStatus(status),
				Messages:  make([]domain.ConversationMessage, 0),
				CreatedAt: createdAt,
				UpdatedAt: updatedAt,
			}

			conversations = append(conversations, conversation)
		}

		return conversations, result.Err()
	})

	if err != nil {
		return nil, fmt.Errorf("failed to find conversations by session: %w", err)
	}

	return result.([]*domain.Conversation), nil
}

// FindConversationsByProject finds conversations by project ID using graph relationships (graph-native)
func (r *GraphConversationRepository) FindConversationsByProject(ctx context.Context, projectID string) ([]*domain.Conversation, error) {
	// Type assert to get Neo4j driver access for direct Cypher queries
	neo4jGraph, ok := r.graph.(*graph.Neo4jGraph)
	if !ok {
		return nil, fmt.Errorf("FindConversationsByProject requires Neo4j graph implementation")
	}

	driver := neo4jGraph.Driver()
	session := driver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	// Cypher query to traverse relationships and find conversations for project
	query := `
		MATCH (p:Project {id: $projectID})<-[:BELONGS_TO]-(c:Conversation)
		RETURN c.id as id, c.status as status, c.created_at as created_at, c.updated_at as updated_at
		ORDER BY c.created_at DESC
	`

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		result, err := tx.Run(ctx, query, map[string]interface{}{
			"projectID": projectID,
		})
		if err != nil {
			return nil, err
		}

		var conversations []*domain.Conversation
		for result.Next(ctx) {
			record := result.Record()

			// Extract conversation properties
			id := record.Values[0].(string)
			status := record.Values[1].(string)
			createdAtStr := record.Values[2].(string)
			updatedAtStr := record.Values[3].(string)

			// Parse timestamps
			createdAt, err := parseTime(createdAtStr)
			if err != nil {
				return nil, fmt.Errorf("failed to parse created_at: %w", err)
			}

			updatedAt, err := parseTime(updatedAtStr)
			if err != nil {
				return nil, fmt.Errorf("failed to parse updated_at: %w", err)
			}

			// Create conversation object
			conversation := &domain.Conversation{
				ID:        id,
				Status:    domain.ConversationStatus(status),
				Messages:  make([]domain.ConversationMessage, 0),
				CreatedAt: createdAt,
				UpdatedAt: updatedAt,
			}

			conversations = append(conversations, conversation)
		}

		return conversations, result.Err()
	})

	if err != nil {
		return nil, fmt.Errorf("failed to find conversations by project: %w", err)
	}

	return result.([]*domain.Conversation), nil
}

// FindActiveConversations finds all active conversations
func (r *GraphConversationRepository) FindActiveConversations(ctx context.Context) ([]*domain.Conversation, error) {
	return r.FindConversationsByStatus(ctx, domain.ConversationStatusActive)
}

// GetAllConversations retrieves all conversations in the system
func (r *GraphConversationRepository) GetAllConversations(ctx context.Context) ([]*domain.Conversation, error) {
	// Query all conversations without any filters
	conversationProps, err := r.graph.QueryNodes(ctx, NodeTypeConversation, map[string]interface{}{})
	if err != nil {
		return nil, fmt.Errorf("failed to query all conversations: %w", err)
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

// mapToConversation converts map properties to Conversation domain object (graph-native)
func (r *GraphConversationRepository) mapToConversation(props map[string]interface{}) (*domain.Conversation, error) {
	id, ok := props["id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid conversation id")
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

	// Graph-native: Create conversation object with only essential properties
	// Relationships (user, session, project, execution plans) are queried via graph traversal, not stored as properties
	conversation := &domain.Conversation{
		ID:        id,
		Status:    domain.ConversationStatus(statusStr),
		Messages:  make([]domain.ConversationMessage, 0), // Messages loaded separately
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
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

	// Handle metadata (may be nil or JSON string)
	metadata := make(map[string]interface{})
	if metadataRaw, exists := props["metadata"]; exists && metadataRaw != nil {
		if metadataJSON, ok := metadataRaw.(string); ok {
			// Metadata is stored as JSON string, deserialize it
			if err := json.Unmarshal([]byte(metadataJSON), &metadata); err != nil {
				// If deserialization fails, keep metadata empty (don't fail the whole operation)
				metadata = make(map[string]interface{})
			}
		} else if metadataMap, ok := metadataRaw.(map[string]interface{}); ok {
			// Backward compatibility for old format (shouldn't happen but just in case)
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

// GetConversationContext retrieves conversation context using graph traversal
// REAL IMPLEMENTATION: Graph traversal with Cypher query
func (r *GraphConversationRepository) GetConversationContext(ctx context.Context, conversationID string) (*domain.ConversationContextData, error) {
	// Type assert to get Neo4j driver access for direct Cypher queries
	neo4jGraph, ok := r.graph.(*graph.Neo4jGraph)
	if !ok {
		return nil, fmt.Errorf("GetConversationContext requires Neo4j graph implementation")
	}

	driver := neo4jGraph.Driver()
	session := driver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	// Cypher query to traverse relationships and get full context
	query := `
		MATCH (c:Conversation {id: $conversationID})
		OPTIONAL MATCH (s:Session)-[:IN_SESSION]->(c)
		OPTIONAL MATCH (u:User)-[:PARTICIPANT_IN]->(c)  
		OPTIONAL MATCH (c)-[:BELONGS_TO]->(p:Project)
		RETURN 
			c.id as conversationID,
			s.id as sessionID,
			u.id as userID,
			p.id as projectID,
			p.name as projectName
	`

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		result, err := tx.Run(ctx, query, map[string]interface{}{
			"conversationID": conversationID,
		})
		if err != nil {
			return nil, err
		}

		if !result.Next(ctx) {
			return nil, fmt.Errorf("conversation not found: %s", conversationID)
		}

		record := result.Record()

		// Extract values, handling nil cases for optional matches
		var sessionID, userID, projectID, projectName string

		if record.Values[1] != nil {
			sessionID = record.Values[1].(string)
		}
		if record.Values[2] != nil {
			userID = record.Values[2].(string)
		}
		if record.Values[3] != nil {
			projectID = record.Values[3].(string)
		}
		if record.Values[4] != nil {
			projectName = record.Values[4].(string)
		}

		return &domain.ConversationContextData{
			ConversationID: conversationID,
			SessionID:      sessionID,
			UserID:         userID,
			ProjectID:      projectID,
			ProjectName:    projectName,
		}, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get conversation context: %w", err)
	}

	return result.(*domain.ConversationContextData), nil
}

// GetConversationWithRelationships retrieves conversation with all related entities using graph traversal
// This implements the graph-native approach for complete context loading in a single query
func (r *GraphConversationRepository) GetConversationWithRelationships(ctx context.Context, conversationID string) (*domain.ConversationWithRelationships, error) {
	// Type assert to get Neo4j driver access for direct Cypher queries
	neo4jGraph, ok := r.graph.(*graph.Neo4jGraph)
	if !ok {
		return nil, fmt.Errorf("GetConversationWithRelationships requires Neo4j graph implementation")
	}

	driver := neo4jGraph.Driver()
	session := driver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	// Comprehensive Cypher query to get conversation with all relationships
	query := `
		MATCH (c:Conversation {id: $conversationID})
		OPTIONAL MATCH (u:User)-[:PARTICIPANT_IN]->(c)
		OPTIONAL MATCH (s:Session)-[:IN_SESSION]->(c)
		OPTIONAL MATCH (c)-[:BELONGS_TO]->(p:Project)
		OPTIONAL MATCH (c)-[:HAS_EXECUTION_PLAN]->(ep:ExecutionPlan)
		RETURN 
			c.id as conversationID,
			c.status as conversationStatus,
			c.created_at as conversationCreatedAt,
			c.updated_at as conversationUpdatedAt,
			u.id as userID,
			u.email as userEmail,
			s.id as sessionID,
			s.status as sessionStatus,
			p.id as projectID,
			p.name as projectName,
			collect(distinct {id: ep.id, status: ep.status}) as executionPlans
	`

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		result, err := tx.Run(ctx, query, map[string]interface{}{
			"conversationID": conversationID,
		})
		if err != nil {
			return nil, err
		}

		if !result.Next(ctx) {
			return nil, fmt.Errorf("conversation not found: %s", conversationID)
		}

		record := result.Record()

		// Extract conversation data
		convID := record.Values[0].(string)
		convStatus := record.Values[1].(string)
		convCreatedAtStr := record.Values[2].(string)
		convUpdatedAtStr := record.Values[3].(string)

		// Parse timestamps
		createdAt, err := parseTime(convCreatedAtStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse conversation created_at: %w", err)
		}

		updatedAt, err := parseTime(convUpdatedAtStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse conversation updated_at: %w", err)
		}

		// Create conversation object
		conversation := &domain.Conversation{
			ID:        convID,
			Status:    domain.ConversationStatus(convStatus),
			Messages:  make([]domain.ConversationMessage, 0), // Messages loaded separately if needed
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		}

		// Extract user data (optional)
		var user *domain.UserInfo
		if record.Values[4] != nil {
			user = &domain.UserInfo{
				ID:    record.Values[4].(string),
				Email: record.Values[5].(string),
			}
		}

		// Extract session data (optional)
		var sessionInfo *domain.SessionInfo
		if record.Values[6] != nil {
			sessionInfo = &domain.SessionInfo{
				ID:     record.Values[6].(string),
				Status: record.Values[7].(string),
			}
		}

		// Extract project data (optional)
		var project *domain.ProjectInfo
		if record.Values[8] != nil {
			project = &domain.ProjectInfo{
				ID:   record.Values[8].(string),
				Name: record.Values[9].(string),
			}
		}

		// Extract execution plans (may be empty list)
		var executionPlans []*domain.ExecutionPlanInfo
		if record.Values[10] != nil {
			executionPlanData := record.Values[10].([]interface{})
			for _, epData := range executionPlanData {
				if epMap, ok := epData.(map[string]interface{}); ok {
					if epMap["id"] != nil && epMap["status"] != nil {
						executionPlans = append(executionPlans, &domain.ExecutionPlanInfo{
							ID:     epMap["id"].(string),
							Status: epMap["status"].(string),
						})
					}
				}
			}
		}

		return &domain.ConversationWithRelationships{
			Conversation:   conversation,
			User:           user,
			Session:        sessionInfo,
			Project:        project,
			ExecutionPlans: executionPlans,
		}, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get conversation with relationships: %w", err)
	}

	return result.(*domain.ConversationWithRelationships), nil
}
