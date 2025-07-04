package infrastructure

import (
	"context"
	"fmt"
	"time"

	"neuromesh/internal/conversation/domain"
	"neuromesh/internal/graph"
	planningdomain "neuromesh/internal/planning/domain"
)

// Constants for graph node types and relationships
const (
	NodeTypeUser         = "User"
	NodeTypeConversation = "Conversation"
	NodeTypeMessage      = "Message"
	NodeTypeUserRequest  = "UserRequest"
	NodeTypeAIDecision   = "AIDecision"

	RelationshipHasMessage  = "HAS_MESSAGE"
	RelationshipStartedBy   = "STARTED_BY"
	RelationshipRequestedBy = "REQUESTED_BY"
	RelationshipDecidedBy   = "DECIDED_BY"
	RelationshipHasDecision = "HAS_DECISION"

	TimeFormat = "2006-01-02T15:04:05Z"
)

// GraphConversationRepository implements conversation repository using the graph backend
type GraphConversationRepository struct {
	graph graph.Graph
}

// NewGraphConversationRepository creates a new graph-based conversation repository
func NewGraphConversationRepository(g graph.Graph) *GraphConversationRepository {
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

// parseIntFromInterface safely parses an integer from an interface{} that could be int, int64, or float64
func parseIntFromInterface(value interface{}, fieldName string) (int, error) {
	switch v := value.(type) {
	case int:
		return v, nil
	case int64:
		return int(v), nil
	case float64:
		return int(v), nil
	default:
		return 0, fmt.Errorf("invalid %s type: %T", fieldName, value)
	}
}

// parseFloatFromInterface safely parses a float64 from an interface{} that could be int, int64, or float64
func parseFloatFromInterface(value interface{}, fieldName string) (float64, error) {
	switch v := value.(type) {
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case float64:
		return v, nil
	default:
		return 0, fmt.Errorf("invalid %s type: %T", fieldName, value)
	}
}

// buildConversationProperties converts Conversation domain object to graph properties
func (r *GraphConversationRepository) buildConversationProperties(conversation *domain.Conversation) map[string]interface{} {
	properties := map[string]interface{}{
		"id":               conversation.ID,
		"user_id":          conversation.UserID,
		"status":           string(conversation.Status),
		"created_at":       formatTime(conversation.CreatedAt),
		"updated_at":       formatTime(conversation.UpdatedAt),
		"last_activity_at": formatTime(conversation.LastActivityAt),
		"title":            conversation.Title,
		"summary":          conversation.Summary,
	}

	// Add optional array fields if present
	if len(conversation.ExecutionPlanIDs) > 0 {
		properties["execution_plan_ids"] = conversation.ExecutionPlanIDs
	}

	if len(conversation.Tags) > 0 {
		properties["tags"] = conversation.Tags
	}

	if len(conversation.Context) > 0 {
		properties["context"] = conversation.Context
	}

	return properties
}

// buildMessageProperties converts ConversationMessage domain object to graph properties
func (r *GraphConversationRepository) buildMessageProperties(conversationID string, message *domain.ConversationMessage) map[string]interface{} {
	properties := map[string]interface{}{
		"id":              message.ID,
		"conversation_id": conversationID,
		"role":            string(message.Role),
		"content":         message.Content,
		"timestamp":       formatTime(message.Timestamp),
	}

	// Add metadata if present
	if len(message.Metadata) > 0 {
		properties["metadata"] = message.Metadata
	}

	return properties
}

// buildRelationshipProperties creates standard relationship properties
func (r *GraphConversationRepository) buildRelationshipProperties() map[string]interface{} {
	return map[string]interface{}{
		"created_at": formatTime(time.Now().UTC()),
	}
}

// buildUserRequestProperties converts UserRequest domain object to graph properties
func (r *GraphConversationRepository) buildUserRequestProperties(userRequest *domain.UserRequest) map[string]interface{} {
	properties := map[string]interface{}{
		"id":              userRequest.ID,
		"user_id":         userRequest.UserID,
		"session_id":      userRequest.SessionID,
		"user_input":      userRequest.UserInput,
		"analyzed_intent": string(userRequest.AnalyzedIntent),
		"category":        string(userRequest.Category),
		"status":          string(userRequest.Status),
		"created_at":      formatTime(userRequest.CreatedAt),
		"updated_at":      formatTime(userRequest.UpdatedAt),
		"confidence":      userRequest.Confidence,
	}

	// Add optional fields
	if userRequest.ConversationID != "" {
		properties["conversation_id"] = userRequest.ConversationID
	}

	if userRequest.ProcessedAt != nil {
		properties["processed_at"] = formatTime(*userRequest.ProcessedAt)
	}

	if userRequest.PreviousRequest != "" {
		properties["previous_request"] = userRequest.PreviousRequest
	}

	if len(userRequest.RequiredAgents) > 0 {
		properties["required_agents"] = userRequest.RequiredAgents
	}

	if len(userRequest.Context) > 0 {
		properties["context"] = userRequest.Context
	}

	return properties
}

// buildAIDecisionProperties converts AIDecision domain object to graph properties
func (r *GraphConversationRepository) buildAIDecisionProperties(aiDecision *planningdomain.AIDecision) map[string]interface{} {
	properties := map[string]interface{}{
		"id":             aiDecision.ID,
		"request_id":     aiDecision.RequestID,
		"decision_type":  string(aiDecision.Type),
		"reasoning":      aiDecision.Reasoning,
		"confidence":     aiDecision.Confidence,
		"status":         string(aiDecision.Status),
		"created_at":     formatTime(aiDecision.CreatedAt),
		"updated_at":     formatTime(aiDecision.UpdatedAt),
		"execution_plan": aiDecision.ExecutionPlan,
	}

	// Add optional fields
	if aiDecision.ConversationID != "" {
		properties["conversation_id"] = aiDecision.ConversationID
	}

	if aiDecision.CompletedAt != nil {
		properties["completed_at"] = formatTime(*aiDecision.CompletedAt)
	}

	if len(aiDecision.SelectedAgents) > 0 {
		properties["selected_agents"] = aiDecision.SelectedAgents
	}

	if len(aiDecision.AgentInstructions) > 0 {
		properties["agent_instructions"] = aiDecision.AgentInstructions
	}

	if len(aiDecision.Context) > 0 {
		properties["context"] = aiDecision.Context
	}

	if len(aiDecision.PreviousDecisions) > 0 {
		properties["previous_decisions"] = aiDecision.PreviousDecisions
	}

	return properties
}

// EnsureConversationSchema ensures that the required schema for Conversation domain is in place
func (r *GraphConversationRepository) EnsureConversationSchema(ctx context.Context) error {
	// Create unique constraint for Conversation nodes
	if err := r.graph.CreateUniqueConstraint(ctx, NodeTypeConversation, "id"); err != nil {
		return fmt.Errorf("failed to create conversation id constraint: %w", err)
	}

	// Create indexes for Conversation nodes
	conversationIndexes := []string{"user_id", "status", "created_at", "last_activity_at"}
	for _, property := range conversationIndexes {
		if err := r.graph.CreateIndex(ctx, NodeTypeConversation, property); err != nil {
			return fmt.Errorf("failed to create conversation %s index: %w", property, err)
		}
	}

	return nil
}

// EnsureMessageSchema ensures that the required schema for Message domain is in place
func (r *GraphConversationRepository) EnsureMessageSchema(ctx context.Context) error {
	// Create unique constraint for Message nodes
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
	properties := r.buildConversationProperties(conversation)

	return r.graph.AddNode(ctx, NodeTypeConversation, conversation.ID, properties)
}

// CreateMessage creates a message node in the graph
func (r *GraphConversationRepository) CreateMessage(ctx context.Context, conversationID string, message *domain.ConversationMessage) error {
	properties := r.buildMessageProperties(conversationID, message)

	return r.graph.AddNode(ctx, NodeTypeMessage, message.ID, properties)
}

// LinkMessageToConversation creates a relationship between message and conversation
func (r *GraphConversationRepository) LinkMessageToConversation(ctx context.Context, messageID, conversationID string) error {
	properties := r.buildRelationshipProperties()

	return r.graph.AddEdge(ctx, NodeTypeConversation, conversationID, NodeTypeMessage, messageID, RelationshipHasMessage, properties)
}

// LinkConversationToUser creates a relationship between conversation and user
func (r *GraphConversationRepository) LinkConversationToUser(ctx context.Context, conversationID, userID string) error {
	properties := r.buildRelationshipProperties()

	return r.graph.AddEdge(ctx, NodeTypeUser, userID, NodeTypeConversation, conversationID, RelationshipStartedBy, properties)
}

// GetConversationWithMessages retrieves a conversation with its messages
func (r *GraphConversationRepository) GetConversationWithMessages(ctx context.Context, conversationID string) (*domain.Conversation, error) {
	// Get the conversation node
	conversationProps, err := r.graph.GetNode(ctx, NodeTypeConversation, conversationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation: %w", err)
	}

	if conversationProps == nil {
		return nil, fmt.Errorf("conversation not found: %s", conversationID)
	}

	// Convert map properties back to Conversation domain object
	conversation, err := r.mapToConversation(conversationProps)
	if err != nil {
		return nil, fmt.Errorf("failed to map conversation properties: %w", err)
	}

	return conversation, nil
}

// GetUserConversations retrieves all conversations for a user
func (r *GraphConversationRepository) GetUserConversations(ctx context.Context, userID string) ([]*domain.Conversation, error) {
	// Query conversations by user_id property
	filters := map[string]interface{}{
		"user_id": userID,
	}

	conversationNodes, err := r.graph.QueryNodes(ctx, NodeTypeConversation, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to query user conversations: %w", err)
	}

	var conversations []*domain.Conversation
	for _, nodeProps := range conversationNodes {
		conversation, err := r.mapToConversation(nodeProps)
		if err != nil {
			return nil, fmt.Errorf("failed to map conversation properties: %w", err)
		}
		conversations = append(conversations, conversation)
	}

	return conversations, nil
}

// RED Phase: UserRequest and AIDecision methods (not implemented yet)
func (r *GraphConversationRepository) EnsureUserRequestSchema(ctx context.Context) error {
	// Create unique constraint for UserRequest nodes
	if err := r.graph.CreateUniqueConstraint(ctx, NodeTypeUserRequest, "id"); err != nil {
		return fmt.Errorf("failed to create user_request id constraint: %w", err)
	}

	// Create indexes for UserRequest nodes
	userRequestIndexes := []string{"user_id", "session_id", "conversation_id", "status", "analyzed_intent", "created_at"}
	for _, property := range userRequestIndexes {
		if err := r.graph.CreateIndex(ctx, NodeTypeUserRequest, property); err != nil {
			return fmt.Errorf("failed to create user_request %s index: %w", property, err)
		}
	}

	return nil
}

func (r *GraphConversationRepository) EnsureAIDecisionSchema(ctx context.Context) error {
	// Create unique constraint for AIDecision nodes
	if err := r.graph.CreateUniqueConstraint(ctx, NodeTypeAIDecision, "id"); err != nil {
		return fmt.Errorf("failed to create ai_decision id constraint: %w", err)
	}

	// Create indexes for AIDecision nodes
	aiDecisionIndexes := []string{"request_id", "decision_type", "status", "confidence", "created_at"}
	for _, property := range aiDecisionIndexes {
		if err := r.graph.CreateIndex(ctx, NodeTypeAIDecision, property); err != nil {
			return fmt.Errorf("failed to create ai_decision %s index: %w", property, err)
		}
	}

	return nil
}

func (r *GraphConversationRepository) CreateUserRequest(ctx context.Context, userRequest *domain.UserRequest) error {
	properties := r.buildUserRequestProperties(userRequest)
	return r.graph.AddNode(ctx, NodeTypeUserRequest, userRequest.ID, properties)
}

func (r *GraphConversationRepository) CreateAIDecision(ctx context.Context, aiDecision *planningdomain.AIDecision) error {
	properties := r.buildAIDecisionProperties(aiDecision)
	return r.graph.AddNode(ctx, NodeTypeAIDecision, aiDecision.ID, properties)
}

func (r *GraphConversationRepository) LinkUserRequestToAIDecision(ctx context.Context, userRequestID, aiDecisionID string) error {
	properties := r.buildRelationshipProperties()
	return r.graph.AddEdge(ctx, NodeTypeUserRequest, userRequestID, NodeTypeAIDecision, aiDecisionID, RelationshipHasDecision, properties)
}

func (r *GraphConversationRepository) GetUserRequestWithDecisions(ctx context.Context, userRequestID string) (*domain.UserRequest, error) {
	// Get the user request node
	userRequestProps, err := r.graph.GetNode(ctx, NodeTypeUserRequest, userRequestID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user request: %w", err)
	}

	if userRequestProps == nil {
		return nil, fmt.Errorf("user request not found: %s", userRequestID)
	}

	// Convert map properties back to UserRequest domain object
	userRequest, err := r.mapToUserRequest(userRequestProps)
	if err != nil {
		return nil, fmt.Errorf("failed to map user request properties: %w", err)
	}

	return userRequest, nil
}

// mapToConversation converts map properties to Conversation domain object
func (r *GraphConversationRepository) mapToConversation(props map[string]interface{}) (*domain.Conversation, error) {
	id, ok := props["id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid conversation id")
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

	lastActivityAtStr, ok := props["last_activity_at"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid last_activity_at")
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

	lastActivityAt, err := parseTime(lastActivityAtStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse last_activity_at: %w", err)
	}

	// Create conversation object
	conversation := &domain.Conversation{
		ID:               id,
		UserID:           userID,
		Status:           domain.ConversationStatus(statusStr),
		Messages:         make([]domain.ConversationMessage, 0), // TODO: Load messages separately
		ExecutionPlanIDs: make([]string, 0),
		CreatedAt:        createdAt,
		UpdatedAt:        updatedAt,
		LastActivityAt:   lastActivityAt,
		Context:          make(map[string]interface{}),
		Tags:             make([]string, 0),
	}

	// Add optional fields
	if title, exists := props["title"]; exists {
		if titleStr, ok := title.(string); ok {
			conversation.Title = titleStr
		}
	}

	if summary, exists := props["summary"]; exists {
		if summaryStr, ok := summary.(string); ok {
			conversation.Summary = summaryStr
		}
	}

	if executionPlanIDs, exists := props["execution_plan_ids"]; exists {
		if planIDs, ok := executionPlanIDs.([]string); ok {
			conversation.ExecutionPlanIDs = planIDs
		}
	}

	if tags, exists := props["tags"]; exists {
		if tagList, ok := tags.([]string); ok {
			conversation.Tags = tagList
		}
	}

	if context, exists := props["context"]; exists {
		if contextMap, ok := context.(map[string]interface{}); ok {
			conversation.Context = contextMap
		}
	}

	return conversation, nil
}

// mapToUserRequest converts map properties to UserRequest domain object
func (r *GraphConversationRepository) mapToUserRequest(props map[string]interface{}) (*domain.UserRequest, error) {
	id, ok := props["id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid user request id")
	}

	userID, ok := props["user_id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid user_id")
	}

	sessionID, ok := props["session_id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid session_id")
	}

	userInput, ok := props["user_input"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid user_input")
	}

	analyzedIntentStr, ok := props["analyzed_intent"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid analyzed_intent")
	}

	categoryStr, ok := props["category"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid category")
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

	// Handle confidence field - Neo4j may return as float64 or int
	confidence, err := parseIntFromInterface(props["confidence"], "confidence")
	if err != nil {
		return nil, err
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

	// Create user request object
	userRequest := &domain.UserRequest{
		ID:             id,
		UserID:         userID,
		SessionID:      sessionID,
		UserInput:      userInput,
		AnalyzedIntent: domain.RequestIntent(analyzedIntentStr),
		Category:       domain.RequestCategory(categoryStr),
		Status:         domain.RequestStatus(statusStr),
		CreatedAt:      createdAt,
		UpdatedAt:      updatedAt,
		Confidence:     confidence,
		RequiredAgents: make([]string, 0),
		Context:        make(map[string]interface{}),
	}

	// Add optional fields
	if conversationID, exists := props["conversation_id"]; exists {
		if convID, ok := conversationID.(string); ok {
			userRequest.ConversationID = convID
		}
	}

	if processedAtStr, exists := props["processed_at"]; exists {
		if processedStr, ok := processedAtStr.(string); ok {
			processedAt, err := parseTime(processedStr)
			if err == nil {
				userRequest.ProcessedAt = &processedAt
			}
		}
	}

	if previousRequest, exists := props["previous_request"]; exists {
		if prevReq, ok := previousRequest.(string); ok {
			userRequest.PreviousRequest = prevReq
		}
	}

	if requiredAgents, exists := props["required_agents"]; exists {
		if agents, ok := requiredAgents.([]string); ok {
			userRequest.RequiredAgents = agents
		}
	}

	if context, exists := props["context"]; exists {
		if contextMap, ok := context.(map[string]interface{}); ok {
			userRequest.Context = contextMap
		}
	}

	return userRequest, nil
}

// mapToAIDecision converts map properties to AIDecision domain object
func (r *GraphConversationRepository) mapToAIDecision(props map[string]interface{}) (*planningdomain.AIDecision, error) {
	id, ok := props["id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid ai decision id")
	}

	requestID, ok := props["request_id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid request_id")
	}

	decisionTypeStr, ok := props["decision_type"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid decision_type")
	}

	reasoning, ok := props["reasoning"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid reasoning")
	}

	confidence, err := parseFloatFromInterface(props["confidence"], "confidence")
	if err != nil {
		return nil, err
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

	executionPlan, ok := props["execution_plan"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid execution_plan")
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

	// Create AI decision object
	aiDecision := &planningdomain.AIDecision{
		ID:                id,
		RequestID:         requestID,
		Type:              domain.DecisionType(decisionTypeStr),
		Reasoning:         reasoning,
		Confidence:        confidence,
		Status:            domain.DecisionStatus(statusStr),
		CreatedAt:         createdAt,
		UpdatedAt:         updatedAt,
		ExecutionPlan:     executionPlan,
		SelectedAgents:    make([]string, 0),
		AgentInstructions: make(map[string]string),
		Context:           make(map[string]interface{}),
		PreviousDecisions: make([]string, 0),
	}

	// Add optional fields
	if conversationID, exists := props["conversation_id"]; exists {
		if convID, ok := conversationID.(string); ok {
			aiDecision.ConversationID = convID
		}
	}

	if completedAtStr, exists := props["completed_at"]; exists {
		if compAtStr, ok := completedAtStr.(string); ok {
			completedAt, err := parseTime(compAtStr)
			if err == nil {
				aiDecision.CompletedAt = &completedAt
			}
		}
	}

	if selectedAgents, exists := props["selected_agents"]; exists {
		if agents, ok := selectedAgents.([]string); ok {
			aiDecision.SelectedAgents = agents
		}
	}

	if agentInstructions, exists := props["agent_instructions"]; exists {
		if instructions, ok := agentInstructions.(map[string]string); ok {
			aiDecision.AgentInstructions = instructions
		}
	}

	if context, exists := props["context"]; exists {
		if contextMap, ok := context.(map[string]interface{}); ok {
			aiDecision.Context = contextMap
		}
	}

	if previousDecisions, exists := props["previous_decisions"]; exists {
		if decisions, ok := previousDecisions.([]string); ok {
			aiDecision.PreviousDecisions = decisions
		}
	}

	return aiDecision, nil
}
