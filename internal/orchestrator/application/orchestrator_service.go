package application

import (
	"context"
	"fmt"
	"strings"

	"neuromesh/internal/logging"
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
	logger               logging.Logger
}

// NewOrchestratorService creates a new orchestrator service implementation
func NewOrchestratorService(
	aiDecisionEngine AIDecisionEngineInterface,
	graphExplorer GraphExplorerInterface,
	aiConversationEngine AIConversationEngineInterface,
	learningService LearningServiceInterface,
	logger logging.Logger,
) *OrchestratorService {
	return &OrchestratorService{
		aiDecisionEngine:     aiDecisionEngine,
		graphExplorer:        graphExplorer,
		aiConversationEngine: aiConversationEngine,
		learningService:      learningService,
		logger:               logger,
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
	ors.logger.Info("🔍 ProcessUserRequest started", "userInput", request.UserInput, "userID", request.UserID)

	// 1. Get agent context for AI decision making
	agentContext, err := ors.graphExplorer.GetAgentContext(ctx)
	if err != nil {
		ors.logger.Error("❌ Failed to get agent context", err)
		return &OrchestratorResult{
			Success: false,
			Error:   fmt.Sprintf("Failed to get agent context: %v", err),
		}, nil // Return result with error, not Go error
	}
	ors.logger.Info("✅ Agent context retrieved", "agentContext", agentContext)

	// 2. Perform AI analysis and decision making
	analysis, err := ors.aiDecisionEngine.ExploreAndAnalyze(ctx, request.UserInput, request.UserID, agentContext)
	if err != nil {
		ors.logger.Error("❌ Failed to analyze request", err)
		return &OrchestratorResult{
			Success: false,
			Error:   fmt.Sprintf("Failed to analyze request: %v", err),
		}, nil
	}
	ors.logger.Info("✅ Analysis completed", "intent", analysis.Intent, "requiredAgents", analysis.RequiredAgents)

	decision, err := ors.aiDecisionEngine.MakeDecision(ctx, request.UserInput, request.UserID, analysis)
	if err != nil {
		ors.logger.Error("❌ Failed to make decision", err)
		return &OrchestratorResult{
			Success: false,
			Error:   fmt.Sprintf("Failed to make decision: %v", err),
		}, nil
	}
	ors.logger.Info("✅ Decision made", "type", decision.Type, "clarificationQuestion", decision.ClarificationQuestion, "executionPlan", decision.ExecutionPlan)

	result := &OrchestratorResult{
		Analysis: analysis,
		Decision: decision,
		Success:  true,
	}

	// 3. Handle decision based on type
	if decision.Type == orchestratorDomain.DecisionTypeClarify {
		ors.logger.Info("🤔 Decision type: Clarify")
		result.Message = decision.ClarificationQuestion
	} else if decision.Type == orchestratorDomain.DecisionTypeExecute {
		ors.logger.Info("🚀 Decision type: Execute", "requiredAgents", len(analysis.RequiredAgents))

		// Check if this is a meta-query that should be handled with AI orchestrator knowledge
		if ors.isOrchestratorMetaQuery(request.UserInput) {
			ors.logger.Info("🏛️ Meta-query detected, using AI to provide intelligent system insights")
			// Use AI conversation engine with orchestrator context for dynamic, intelligent responses
			result.Message = ors.handleMetaQuery(ctx, request.UserInput, agentContext)
		} else if len(analysis.RequiredAgents) > 0 {
			// AI-native execution: Let AI orchestrate with agents for actual user tasks
			ors.logger.Info("🤖 Using AI conversation engine with agents", "agents", analysis.RequiredAgents)
			// Use injected AI conversation engine for agent coordination
			aiResult, err := ors.aiConversationEngine.ProcessWithAgents(ctx, request.UserInput, request.UserID, agentContext)
			if err != nil {
				ors.logger.Error("❌ AI-native execution failed", err)
				result.Success = false
				result.Error = fmt.Sprintf("AI-native execution failed: %v", err)
			} else {
				ors.logger.Info("✅ AI conversation engine result", "aiResult", aiResult)
				result.Message = aiResult
			}
		} else {
			ors.logger.Info("📝 No agents required, using execution plan")
			result.Message = decision.ExecutionPlan
		}
	} else {
		ors.logger.Warn("❓ Unknown decision type", "type", decision.Type)
	}

	ors.logger.Info("✅ Final result", "success", result.Success, "message", result.Message, "error", result.Error)

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

// isOrchestratorMetaQuery detects if a user input is a meta-query about the orchestrator system
// that should be answered directly rather than routed through agents
func (ors *OrchestratorService) isOrchestratorMetaQuery(userInput string) bool {
	lowercaseInput := strings.ToLower(userInput)

	// Define meta-query patterns that should be handled directly by orchestrator
	metaQueryPatterns := []string{
		"what agents",
		"list agents",
		"show agents",
		"available agents",
		"agent capabilities",
		"system status",
		"orchestrator status",
		"are you healthy",
		"health check",
		"what can you do",
		"help",
		"how do you work",
		"what is your purpose",
	}

	for _, pattern := range metaQueryPatterns {
		if strings.Contains(lowercaseInput, pattern) {
			return true
		}
	}

	return false
}

// handleMetaQuery uses AI to provide intelligent, dynamic responses to orchestrator meta-queries
// This replaces hardcoded responses with AI-native understanding of the system state
func (ors *OrchestratorService) handleMetaQuery(ctx context.Context, userInput, agentContext string) string {
	// Use AI conversation engine to provide intelligent, context-aware responses
	// The AI understands the current system state and can provide much more dynamic answers
	systemPrompt := fmt.Sprintf(`You are the NeuroMesh AI Orchestrator responding to a meta-query about the system.

Current System State:
%s

User Query: %s

Provide a helpful, accurate response about the orchestrator system, available agents, capabilities, or system status. 
Be conversational and informative. If agents are available, describe their capabilities dynamically.
If no agents are available, explain what the orchestrator can do and how agents would enhance the system.`,
		agentContext, userInput)

	// Try AI-powered response first
	aiResult, err := ors.aiConversationEngine.ProcessWithAgents(ctx, systemPrompt, "orchestrator-meta", agentContext)
	if err != nil {
		ors.logger.Error("❌ AI meta-query processing failed, using fallback", err)
	}

	return aiResult
}
