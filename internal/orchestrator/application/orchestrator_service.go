package application

import (
	"context"
	"fmt"

	orchestratorDomain "neuromesh/internal/orchestrator/domain"
)

// AIDecisionEngineInterface defines the interface for AI decision making
type AIDecisionEngineInterface interface {
	ExploreAndAnalyze(ctx context.Context, userInput, userID, agentContext string) (*orchestratorDomain.Analysis, error)
	MakeDecision(ctx context.Context, userInput, userID string, analysis *orchestratorDomain.Analysis) (*orchestratorDomain.Decision, error)
}

// GraphExplorerInterface defines the interface for graph exploration
type GraphExplorerInterface interface {
	GetAgentContext(ctx context.Context) (string, error)
}

// AIConversationEngineInterface defines the interface for AI-native conversation orchestration
type AIConversationEngineInterface interface {
	ProcessWithAgents(ctx context.Context, userInput, userID, agentContext string) (string, error)
}

// LearningServiceInterface defines the interface for learning service
type LearningServiceInterface interface {
	StoreInsights(ctx context.Context, userRequest string, analysis *orchestratorDomain.Analysis, decision *orchestratorDomain.Decision) error
	AnalyzePatterns(ctx context.Context, sessionID string) (*orchestratorDomain.ConversationPattern, error)
}

// OrchestratorService represents the clean AI orchestrator service implementation
// This replaces the old ProcessRequest() functionality with clean architecture
type OrchestratorService struct {
	aiDecisionEngine     AIDecisionEngineInterface
	graphExplorer        GraphExplorerInterface
	aiConversationEngine AIConversationEngineInterface
	learningService      LearningServiceInterface
}

// NewOrchestratorService creates a new orchestrator service implementation
func NewOrchestratorService(
	aiDecisionEngine AIDecisionEngineInterface,
	graphExplorer GraphExplorerInterface,
	aiConversationEngine AIConversationEngineInterface,
	learningService LearningServiceInterface,
) *OrchestratorService {
	return &OrchestratorService{
		aiDecisionEngine:     aiDecisionEngine,
		graphExplorer:        graphExplorer,
		aiConversationEngine: aiConversationEngine,
		learningService:      learningService,
	}
}

// OrchestratorRequest represents a user request to the orchestrator
type OrchestratorRequest struct {
	UserInput string `json:"user_input"`
	UserID    string `json:"user_id"`
	SessionID string `json:"session_id,omitempty"`
}

// OrchestratorResult represents the orchestrator's response
type OrchestratorResult struct {
	Message         string                       `json:"message"`
	Decision        *orchestratorDomain.Decision `json:"decision"`
	Analysis        *orchestratorDomain.Analysis `json:"analysis"`
	ExecutionPlanID string                       `json:"execution_plan_id,omitempty"`
	Success         bool                         `json:"success"`
	Error           string                       `json:"error,omitempty"`
}

// ProcessUserRequest is the main entry point that replaces the old ProcessRequest()
// This follows the clean architecture pattern with proper domain boundaries
func (ors *OrchestratorService) ProcessUserRequest(ctx context.Context, request *OrchestratorRequest) (*OrchestratorResult, error) {
	// 1. Get agent context for AI decision making
	agentContext, err := ors.graphExplorer.GetAgentContext(ctx)
	if err != nil {
		return &OrchestratorResult{
			Success: false,
			Error:   fmt.Sprintf("Failed to get agent context: %v", err),
		}, nil // Return result with error, not Go error
	}

	// 2. Perform AI analysis and decision making
	analysis, err := ors.aiDecisionEngine.ExploreAndAnalyze(ctx, request.UserInput, request.UserID, agentContext)
	if err != nil {
		return &OrchestratorResult{
			Success: false,
			Error:   fmt.Sprintf("Failed to analyze request: %v", err),
		}, nil
	}

	decision, err := ors.aiDecisionEngine.MakeDecision(ctx, request.UserInput, request.UserID, analysis)
	if err != nil {
		return &OrchestratorResult{
			Success: false,
			Error:   fmt.Sprintf("Failed to make decision: %v", err),
		}, nil
	}

	result := &OrchestratorResult{
		Analysis: analysis,
		Decision: decision,
		Success:  true,
	}

	// 3. Handle decision based on type
	if decision.Type == orchestratorDomain.DecisionTypeClarify {
		result.Message = decision.ClarificationQuestion
	} else if decision.Type == orchestratorDomain.DecisionTypeExecute {
		// AI-native execution: Let AI orchestrate with agents
		if len(analysis.RequiredAgents) > 0 {
			// Use injected AI conversation engine for agent coordination
			aiResult, err := ors.aiConversationEngine.ProcessWithAgents(ctx, request.UserInput, request.UserID, agentContext)
			if err != nil {
				result.Success = false
				result.Error = fmt.Sprintf("AI-native execution failed: %v", err)
			} else {
				result.Message = aiResult
			}
		} else {
			result.Message = decision.ExecutionPlan
		}
	}

	// 4. Store insights for learning (fixing the architecture violation)
	err = ors.learningService.StoreInsights(ctx, request.UserInput, analysis, decision)
	if err != nil {
		// Log error but don't fail the request
		// In a real implementation, this would be logged properly
		fmt.Printf("Warning: Failed to store insights: %v\n", err)
	}

	return result, nil
}

// ProcessConversation directly uses AI conversation engine for simple requests
func (ors *OrchestratorService) ProcessConversation(ctx context.Context, userInput, userID string) (string, error) {
	agentContext, err := ors.graphExplorer.GetAgentContext(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get agent context: %w", err)
	}

	return ors.aiConversationEngine.ProcessWithAgents(ctx, userInput, userID, agentContext)
}

// AnalyzeConversationPatterns analyzes patterns in user conversations
func (ors *OrchestratorService) AnalyzeConversationPatterns(ctx context.Context, sessionID string) (*orchestratorDomain.ConversationPattern, error) {
	return ors.learningService.AnalyzePatterns(ctx, sessionID)
}
