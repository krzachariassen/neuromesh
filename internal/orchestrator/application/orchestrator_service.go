package application

import (
	"context"
	"fmt"
	"strings"

	executionDomain "neuromesh/internal/execution/domain"
	"neuromesh/internal/logging"
	planningDomain "neuromesh/internal/planning/domain"
)

// AIPlanningEngineInterface defines the interface for unified planning approach
type AIPlanningEngineInterface interface {
	CreateExecutionPlan(ctx context.Context, userInput, userID, agentContext, requestID string) (*planningDomain.PlanningResult, error)
	LinkPlanningResultToConversation(ctx context.Context, planningResultID, conversationID string) error
}

// GraphExplorerInterface defines the interface for graph exploration
type GraphExplorerInterface interface {
	GetAgentContext(ctx context.Context) (string, error)
}

// AIExecutionEngineInterface defines the interface for AI-native execution orchestration
type AIExecutionEngineInterface interface {
	ExecuteWithAgents(ctx context.Context, executionPlan, userInput, userID, agentContext string) (string, error)
}

// AIConversationEngineInterface defines the interface for AI-native conversation orchestration
type AIConversationEngineInterface interface {
	ProcessWithAgents(ctx context.Context, userInput, userID, agentContext string) (string, error)
}

// ConversationServiceInterface defines the interface for conversation management
type ConversationServiceInterface interface {
	LinkExecutionPlan(ctx context.Context, conversationID, planID string) error
}

// NOTE: LearningServiceInterface removed - following YAGNI principles
// Will be re-implemented when learning features are actually needed

// OrchestratorService represents the clean AI orchestrator service implementation
// This replaces the old ProcessRequest() functionality with clean architecture
type OrchestratorService struct {
	aiPlanningEngine    AIPlanningEngineInterface
	graphExplorer       GraphExplorerInterface
	aiExecutionEngine   AIExecutionEngineInterface
	conversationService ConversationServiceInterface
	resultSynthesizer   executionDomain.ResultSynthesizer
	repository          planningDomain.ExecutionPlanRepository
	logger              logging.Logger
}

// NewOrchestratorService creates a new orchestrator service implementation
func NewOrchestratorService(
	aiPlanningEngine AIPlanningEngineInterface,
	graphExplorer GraphExplorerInterface,
	aiExecutionEngine AIExecutionEngineInterface,
	conversationService ConversationServiceInterface,
	resultSynthesizer executionDomain.ResultSynthesizer,
	repository planningDomain.ExecutionPlanRepository,
	logger logging.Logger,
) *OrchestratorService {
	return &OrchestratorService{
		aiPlanningEngine:    aiPlanningEngine,
		graphExplorer:       graphExplorer,
		aiExecutionEngine:   aiExecutionEngine,
		conversationService: conversationService,
		resultSynthesizer:   resultSynthesizer,
		repository:          repository,
		logger:              logger,
	}
}

// OrchestratorRequest represents a user request to the orchestrator
type OrchestratorRequest struct {
	UserInput      string `json:"user_input"`
	UserID         string `json:"user_id"`
	SessionID      string `json:"session_id,omitempty"`
	MessageID      string `json:"message_id,omitempty"`      // ID of the user message that triggered this request
	ConversationID string `json:"conversation_id,omitempty"` // ID of the conversation containing this message
}

// OrchestratorResult represents the orchestrator's response
type OrchestratorResult struct {
	Message         string                         `json:"message"`
	PlanningResult  *planningDomain.PlanningResult `json:"planning_result,omitempty"` // New unified planning result
	ExecutionPlanID string                         `json:"execution_plan_id,omitempty"`
	Success         bool                           `json:"success"`
	Error           string                         `json:"error,omitempty"`
}

// ProcessUserRequest is the main entry point that replaces the old ProcessRequest()
// This follows the clean architecture pattern with proper domain boundaries
func (ors *OrchestratorService) ProcessUserRequest(ctx context.Context, request *OrchestratorRequest) (*OrchestratorResult, error) {
	// 1. Get agent context for AI planning
	agentContext, err := ors.graphExplorer.GetAgentContext(ctx)
	if err != nil {
		return &OrchestratorResult{
			Success: false,
			Error:   fmt.Sprintf("Failed to get agent context: %v", err),
		}, nil // Return result with error, not Go error
	}

	// 2. Perform unified AI planning
	planningResult, err := ors.aiPlanningEngine.CreateExecutionPlan(ctx, request.UserInput, request.UserID, agentContext, request.MessageID)
	if err != nil {
		return &OrchestratorResult{
			Success: false,
			Error:   fmt.Sprintf("Failed to create execution plan: %v", err),
		}, nil
	}

	result := &OrchestratorResult{
		PlanningResult: planningResult,
		Success:        true,
	}

	// 3. Handle planning result based on type
	if planningResult.Type == planningDomain.PlanningTypeClarify {
		ors.logger.Info("🤔 Planning type: Clarify")
		result.Message = planningResult.ClarificationQuestion
	} else if planningResult.Type == planningDomain.PlanningTypeExecute {
		ors.logger.Info("🚀 Planning type: Execute", "requiredAgents", len(planningResult.RequiredAgents))
		result.ExecutionPlanID = planningResult.ExecutionPlanID

		// Check if this is a meta-query that should be handled with AI orchestrator knowledge
		if ors.isOrchestratorMetaQuery(request.UserInput) {
			ors.logger.Info("🏛️ Meta-query detected, using AI to provide intelligent system insights")
			result.Message = ors.handleMetaQuery(ctx, request.UserInput, agentContext)
		} else if len(planningResult.RequiredAgents) > 0 {
			// AI-native execution: Use dedicated execution engine for agent coordination
			ors.logger.Info("🚀 Using AI execution engine with agents", "agents", planningResult.RequiredAgents)

			// Use the execution plan ID as the plan text for now
			executionPlan := planningResult.ExecutionPlanID
			if executionPlan == "" {
				executionPlan = "No execution plan available"
			}

			// Use injected AI execution engine for agent coordination
			executionResult, err := ors.aiExecutionEngine.ExecuteWithAgents(ctx, executionPlan, request.UserInput, request.UserID, agentContext)
			if err != nil {
				ors.logger.Error("❌ AI-native execution failed", err)
				result.Success = false
				result.Error = fmt.Sprintf("AI-native execution failed: %v", err)
			} else {
				ors.logger.Info("✅ AI execution engine result", "executionResult", executionResult)
				result.Message = executionResult
			}
		} else {
			ors.logger.Info("📝 No agents required, using execution plan")
			result.Message = planningResult.ExecutionPlanID
		}
	} else if planningResult.Type == planningDomain.PlanningTypeRespondDirectly {
		ors.logger.Info("💬 Planning type: Respond Directly")
		result.Message = planningResult.DirectResponse
	} else {
		ors.logger.Warn("❓ Unknown planning type", "type", planningResult.Type)
	}

	ors.logger.Info("✅ Final result", "success", result.Success, "message", result.Message, "error", result.Error)

	// 4. Cross-domain coordination: Link execution plans to conversations
	if planningResult.Type == planningDomain.PlanningTypeExecute && planningResult.ExecutionPlanID != "" && request.ConversationID != "" {
		if err := ors.coordinateCrossDomainRelationships(ctx, request.ConversationID, planningResult.ExecutionPlanID); err != nil {
			ors.logger.Warn("Failed to coordinate cross-domain relationships", "error", err, "conversationID", request.ConversationID, "planID", planningResult.ExecutionPlanID)
			// Don't fail the entire request for relationship linking issues
		} else {
			ors.logger.Info("✅ Cross-domain relationship created", "conversationID", request.ConversationID, "planID", planningResult.ExecutionPlanID)
		}
	}

	// 5. Link planning result to conversation for graph visualization
	// This ensures all planning results appear in the conversation graph, regardless of type
	if request.ConversationID != "" && ors.aiPlanningEngine != nil {
		if err := ors.linkPlanningResultToConversation(ctx, planningResult.ID, request.ConversationID); err != nil {
			ors.logger.Warn("Failed to link planning result to conversation", "error", err, "conversationID", request.ConversationID, "planningResultID", planningResult.ID)
			// Don't fail the entire request for relationship linking issues
		} else {
			ors.logger.Info("✅ Planning result linked to conversation", "conversationID", request.ConversationID, "planningResultID", planningResult.ID)
		}
	}

	return result, nil
}

// coordinateCrossDomainRelationships handles cross-domain relationships per clean architecture
// The orchestrator is responsible for coordinating relationships between domains
func (ors *OrchestratorService) coordinateCrossDomainRelationships(ctx context.Context, conversationID, executionPlanID string) error {
	// Use conversation service to link conversation to execution plan
	// This follows clean architecture - orchestrator coordinates, conversation service implements
	if ors.conversationService == nil {
		return fmt.Errorf("conversation service not available for cross-domain coordination")
	}

	return ors.conversationService.LinkExecutionPlan(ctx, conversationID, executionPlanID)
}

// linkPlanningResultToConversation links a planning result to a conversation for graph visualization
// This ensures planning results are visible in conversation graphs regardless of their type
func (ors *OrchestratorService) linkPlanningResultToConversation(ctx context.Context, planningResultID, conversationID string) error {
	// Access the planning result repository through the AI planning engine
	// This follows clean architecture - orchestrator coordinates, planning domain implements
	if ors.aiPlanningEngine == nil {
		return fmt.Errorf("AI planning engine not available for planning result linking")
	}

	return ors.aiPlanningEngine.LinkPlanningResultToConversation(ctx, planningResultID, conversationID)
}

// NOTE: ProcessConversation and AnalyzeConversationPatterns methods removed
// Following YAGNI principles - we're not implementing these features yet

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

// handleMetaQuery provides simple responses to orchestrator meta-queries
// Following YAGNI - keeping it simple for now
func (ors *OrchestratorService) handleMetaQuery(ctx context.Context, userInput, agentContext string) string {
	// Simple implementation for now
	return fmt.Sprintf("This is a meta-query about the orchestrator system. Available agents: %s", agentContext)
}

// ProcessWithSynthesis processes a request and synthesizes results from an execution plan
func (ors *OrchestratorService) ProcessWithSynthesis(ctx context.Context, planID, userInput, userID string) (string, error) {
	if ors.resultSynthesizer == nil {
		return "", fmt.Errorf("result synthesizer not configured")
	}

	// Use the result synthesizer to synthesize agent results
	synthesizedResult, err := ors.resultSynthesizer.SynthesizeResults(ctx, planID)
	if err != nil {
		return "", fmt.Errorf("failed to synthesize results for plan %s: %w", planID, err)
	}

	return synthesizedResult, nil
}

// IsExecutionComplete checks if all steps in an execution plan are complete
func (ors *OrchestratorService) IsExecutionComplete(ctx context.Context, planID string) (bool, error) {
	if ors.repository == nil {
		return false, fmt.Errorf("repository not configured")
	}

	// Get the execution plan
	plan, err := ors.repository.GetByID(ctx, planID)
	if err != nil {
		return false, fmt.Errorf("failed to get execution plan %s: %w", planID, err)
	}

	// Check if all steps are completed
	for _, step := range plan.Steps {
		if step.Status != planningDomain.ExecutionStepStatusCompleted {
			return false, nil
		}
	}

	return true, nil
}
