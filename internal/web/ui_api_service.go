package web

import (
	"context"
	"fmt"

	conversationApp "neuromesh/internal/conversation/application"
	"neuromesh/internal/graph"
	userApp "neuromesh/internal/user/application"
)

// UIAPIService provides clean separation between HTTP handlers and business logic
// Following Single Responsibility Principle: this service only handles UI data transformation
type UIAPIService struct {
	conversationService conversationApp.ConversationService
	userService         userApp.UserService
	graph               graph.Graph // Add graph dependency for real data queries
}

// NewUIAPIService creates a new UI API service
func NewUIAPIService(
	conversationService conversationApp.ConversationService,
	userService userApp.UserService,
) *UIAPIService {
	return &UIAPIService{
		conversationService: conversationService,
		userService:         userService,
		graph:               nil, // Will be set separately for testing
	}
}

// NewUIAPIServiceWithGraph creates a new UI API service with graph dependency
func NewUIAPIServiceWithGraph(
	conversationService conversationApp.ConversationService,
	userService userApp.UserService,
	graph graph.Graph,
) *UIAPIService {
	return &UIAPIService{
		conversationService: conversationService,
		userService:         userService,
		graph:               graph,
	}
}

// GetGraphData retrieves graph data for visualization
// Following clean architecture: business logic separate from HTTP concerns
func (s *UIAPIService) GetGraphData(ctx context.Context, conversationID string) (*GraphDataResponse, error) {
	// TDD GREEN: Implement real data integration to make tests pass
	if s.graph == nil {
		// Fallback to mock data for backward compatibility
		return s.getMockGraphData(conversationID), nil
	}

	// Query conversation node from graph
	conversationNode, err := s.graph.GetNode(ctx, "Conversation", conversationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation node: %w", err)
	}

	// REFACTOR: Extract graph data assembly into helper methods
	graphData := &GraphDataResponse{
		ConversationID: conversationID,
		Nodes:          []GraphNode{},
		Edges:          []GraphEdge{},
	}

	// Add conversation node
	s.addConversationNode(conversationNode, conversationID, graphData)

	// Add user node and edge if available
	if err := s.addUserNodeAndEdge(ctx, conversationNode, conversationID, graphData); err != nil {
		// Log error but continue - user data is optional for graph visualization
	}

	// Add execution plan nodes and edges
	if err := s.addExecutionPlanNodes(ctx, conversationID, graphData); err != nil {
		// Log error but continue - execution plans are optional
	}

	return graphData, nil
}

// REFACTOR: Extract helper methods for cleaner code organization
func (s *UIAPIService) addConversationNode(conversationNode map[string]interface{}, conversationID string, graphData *GraphDataResponse) {
	graphData.Nodes = append(graphData.Nodes, GraphNode{
		ID:   conversationID,
		Type: "conversation",
		Data: map[string]interface{}{
			"title":      conversationNode["id"],
			"status":     conversationNode["status"],
			"created_at": conversationNode["created_at"],
		},
		Position: &NodePosition{X: 300, Y: 200},
	})
}

func (s *UIAPIService) addUserNodeAndEdge(ctx context.Context, conversationNode map[string]interface{}, conversationID string, graphData *GraphDataResponse) error {
	userID, ok := conversationNode["user_id"].(string)
	if !ok || userID == "" {
		return fmt.Errorf("no user_id found in conversation")
	}

	userNodes, err := s.graph.QueryNodes(ctx, "User", map[string]interface{}{"id": userID})
	if err != nil || len(userNodes) == 0 {
		return fmt.Errorf("failed to get user node: %w", err)
	}

	userData := userNodes[0]
	graphData.Nodes = append(graphData.Nodes, GraphNode{
		ID:   userID,
		Type: "user",
		Data: map[string]interface{}{
			"name": userData["name"],
			"id":   userData["id"],
		},
		Position: &NodePosition{X: 100, Y: 200},
	})

	// Add edge between user and conversation
	graphData.Edges = append(graphData.Edges, GraphEdge{
		ID:     fmt.Sprintf("user-%s-conv-%s", userID, conversationID),
		Source: userID,
		Target: conversationID,
		Type:   "created",
	})

	return nil
}

func (s *UIAPIService) addExecutionPlanNodes(ctx context.Context, conversationID string, graphData *GraphDataResponse) error {
	planNodes, err := s.graph.QueryNodes(ctx, "ExecutionPlan", map[string]interface{}{
		"conversation_id": conversationID,
	})
	if err != nil {
		return fmt.Errorf("failed to get execution plans: %w", err)
	}

	for i, planData := range planNodes {
		planID := fmt.Sprintf("%v", planData["id"])
		graphData.Nodes = append(graphData.Nodes, GraphNode{
			ID:   planID,
			Type: "execution_plan",
			Data: map[string]interface{}{
				"name":   planData["name"],
				"status": planData["status"],
			},
			Position: &NodePosition{X: 500, Y: 100 + float64(i*150)},
		})

		// Add edge from conversation to execution plan
		graphData.Edges = append(graphData.Edges, GraphEdge{
			ID:     fmt.Sprintf("conv-%s-plan-%s", conversationID, planID),
			Source: conversationID,
			Target: planID,
			Type:   "linked_to",
		})
	}

	return nil
}

// getMockGraphData returns mock data for backward compatibility
func (s *UIAPIService) getMockGraphData(conversationID string) *GraphDataResponse {
	return &GraphDataResponse{
		ConversationID: conversationID,
		Nodes: []GraphNode{
			{
				ID:       "user-1",
				Type:     "user",
				Data:     map[string]interface{}{"name": "Test User"},
				Position: &NodePosition{X: 100, Y: 200},
			},
			{
				ID:       conversationID,
				Type:     "conversation",
				Data:     map[string]interface{}{"title": "Test Conversation"},
				Position: &NodePosition{X: 300, Y: 200},
			},
		},
		Edges: []GraphEdge{
			{
				ID:     "edge-1",
				Source: "user-1",
				Target: conversationID,
				Type:   "created",
			},
		},
	}
}

// GetExecutionPlan retrieves execution plan data
func (s *UIAPIService) GetExecutionPlan(ctx context.Context, planID string) (*ExecutionPlanResponse, error) {
	// TDD GREEN: Implement real execution plan data to make tests pass
	if s.graph == nil {
		// Fallback to mock data for backward compatibility
		return s.getMockExecutionPlan(planID), nil
	}

	// Query execution plan from graph
	planNode, err := s.graph.GetNode(ctx, "ExecutionPlan", planID)
	if err != nil {
		return nil, fmt.Errorf("failed to get execution plan: %w", err)
	}

	// REFACTOR: Extract step assembly into helper method
	steps, err := s.getExecutionSteps(ctx, planID)
	if err != nil {
		return nil, fmt.Errorf("failed to get execution steps: %w", err)
	}

	planData := &ExecutionPlanResponse{
		ID:          planID,
		Name:        s.safeStringValue(planNode["name"]),
		Description: s.safeStringValue(planNode["description"]),
		Status:      s.safeStringValue(planNode["status"]),
		CreatedAt:   s.safeStringValue(planNode["created_at"]),
		Steps:       steps,
	}

	return planData, nil
}

// REFACTOR: Extract helper methods for cleaner code
func (s *UIAPIService) getExecutionSteps(ctx context.Context, planID string) ([]ExecutionStepData, error) {
	stepNodes, err := s.graph.QueryNodes(ctx, "ExecutionStep", map[string]interface{}{
		"plan_id": planID,
	})
	if err != nil {
		return nil, err
	}

	var steps []ExecutionStepData
	for i, stepData := range stepNodes {
		var completedAt *string
		if stepData["completed_at"] != nil {
			completedAtStr := s.safeStringValue(stepData["completed_at"])
			completedAt = &completedAtStr
		}

		steps = append(steps, ExecutionStepData{
			StepNumber:  i + 1,
			Name:        s.safeStringValue(stepData["name"]),
			Description: s.safeStringValue(stepData["description"]),
			AgentName:   s.safeStringValue(stepData["agent_name"]),
			Status:      s.safeStringValue(stepData["status"]),
			CompletedAt: completedAt,
		})
	}

	return steps, nil
}

// REFACTOR: Add utility method for safe string conversion
func (s *UIAPIService) safeStringValue(value interface{}) string {
	if value == nil {
		return ""
	}
	return fmt.Sprintf("%v", value)
}

// getMockExecutionPlan returns mock data for backward compatibility
func (s *UIAPIService) getMockExecutionPlan(planID string) *ExecutionPlanResponse {
	return &ExecutionPlanResponse{
		ID:          planID,
		Name:        "Test Plan",
		Description: "Test execution plan",
		Status:      "PENDING",
		CreatedAt:   "2025-07-26T10:00:00Z",
		Steps: []ExecutionStepData{
			{
				StepNumber:  1,
				Name:        "First Step",
				Description: "Execute first action",
				AgentName:   "text-processor",
				Status:      "PENDING",
				CompletedAt: nil,
			},
		},
	}
}

// GetConversationHistory retrieves conversation history
func (s *UIAPIService) GetConversationHistory(ctx context.Context, sessionID string) (*ConversationHistoryResponse, error) {
	// TDD GREEN: Implement real conversation history to make tests pass
	if s.graph == nil {
		// Fallback to mock data for backward compatibility
		return s.getMockConversationHistory(sessionID), nil
	}

	// REFACTOR: Extract conversation and message assembly into helper methods
	conversations, err := s.getConversationsBySession(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversations: %w", err)
	}

	messages, err := s.getMessagesByConversations(ctx, conversations)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}

	historyData := &ConversationHistoryResponse{
		SessionID:     sessionID,
		Conversations: conversations,
		Messages:      messages,
	}

	return historyData, nil
}

// REFACTOR: Extract helper methods for better organization
func (s *UIAPIService) getConversationsBySession(ctx context.Context, sessionID string) ([]ConversationData, error) {
	conversationNodes, err := s.graph.QueryNodes(ctx, "Conversation", map[string]interface{}{
		"session_id": sessionID,
	})
	if err != nil {
		return nil, err
	}

	var conversations []ConversationData
	for _, convNode := range conversationNodes {
		conversations = append(conversations, ConversationData{
			ID:        s.safeStringValue(convNode["id"]),
			SessionID: sessionID,
			UserID:    s.safeStringValue(convNode["user_id"]),
			Status:    s.safeStringValue(convNode["status"]),
			CreatedAt: s.safeStringValue(convNode["created_at"]),
		})
	}

	return conversations, nil
}

func (s *UIAPIService) getMessagesByConversations(ctx context.Context, conversations []ConversationData) ([]MessageData, error) {
	var allMessages []MessageData

	for _, conv := range conversations {
		messageNodes, err := s.graph.QueryNodes(ctx, "ConversationMessage", map[string]interface{}{
			"conversation_id": conv.ID,
		})
		if err != nil {
			// Log error but continue with other conversations
			continue
		}

		for _, msgNode := range messageNodes {
			allMessages = append(allMessages, MessageData{
				ID:             s.safeStringValue(msgNode["id"]),
				ConversationID: conv.ID,
				Role:           s.safeStringValue(msgNode["role"]),
				Content:        s.safeStringValue(msgNode["content"]),
				CreatedAt:      s.safeStringValue(msgNode["timestamp"]),
			})
		}
	}

	return allMessages, nil
}

// getMockConversationHistory returns mock data for backward compatibility
func (s *UIAPIService) getMockConversationHistory(sessionID string) *ConversationHistoryResponse {
	return &ConversationHistoryResponse{
		SessionID: sessionID,
		Conversations: []ConversationData{
			{
				ID:        "conv-123",
				SessionID: sessionID,
				UserID:    "user-1",
				Status:    "active",
				CreatedAt: "2025-07-26T10:00:00Z",
			},
		},
		Messages: []MessageData{
			{
				ID:             "msg-1",
				ConversationID: "conv-123",
				Role:           "user",
				Content:        "Hello",
				CreatedAt:      "2025-07-26T10:01:00Z",
			},
			{
				ID:             "msg-2",
				ConversationID: "conv-123",
				Role:           "assistant",
				Content:        "Hi there!",
				CreatedAt:      "2025-07-26T10:01:30Z",
			},
		},
	}
}

// GetAgentStatus retrieves agent status information
func (s *UIAPIService) GetAgentStatus(ctx context.Context) (*AgentStatusResponse, error) {
	// TODO: Phase 1 implementation - integrate with agent registry

	// For MVP, return structured mock data
	agentStatus := &AgentStatusResponse{
		Agents: []AgentData{
			{
				Name:         "text-processor",
				Type:         "processing",
				Status:       "active",
				Capabilities: []string{"text_analysis", "nlp_processing"},
				Metadata: map[string]interface{}{
					"last_active": "2025-07-26T10:00:00Z",
				},
			},
			{
				Name:         "data-analyzer",
				Type:         "analysis",
				Status:       "active",
				Capabilities: []string{"data_processing", "statistical_analysis"},
				Metadata: map[string]interface{}{
					"last_active": "2025-07-26T09:58:00Z",
				},
			},
		},
	}

	return agentStatus, nil
}
