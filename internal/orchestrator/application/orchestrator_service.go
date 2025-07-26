package application

import (
	"context"
	"fmt"
	"strings"

	executionDomain "neuromesh/internal/execution/domain"
	"neuromesh/internal/logging"
	planningDomain "neuromesh/internal/planning/domain"
)

// AIDecisionEngineInterface defines the interface for AI decision making
type AIDecisionEngineInterface interface {
	ExploreAndAnalyze(ctx context.Context, userInput, userID, agentContext, requestID string) (*planningDomain.Analysis, error)
	MakeDecision(ctx context.Context, userInput, userID string, analysis *planningDomain.Analysis, requestID string) (*planningDomain.Decision, error)
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

// LearningServiceInterface defines the interface for learning service
type LearningServiceInterface interface {
	StoreInsights(ctx context.Context, userRequest string, analysis *planningDomain.Analysis, decision *planningDomain.Decision) error
	AnalyzePatterns(ctx context.Context, sessionID string) error // Simplified for now, remove ConversationPattern reference
}

// OrchestratorService represents the clean AI orchestrator service implementation
// This replaces the old ProcessRequest() functionality with clean architecture
type OrchestratorService struct {
	aiDecisionEngine  AIDecisionEngineInterface
	graphExplorer     GraphExplorerInterface
	aiExecutionEngine AIExecutionEngineInterface
	resultSynthesizer executionDomain.ResultSynthesizer
	repository        planningDomain.ExecutionPlanRepository
	logger            logging.Logger
}

// NewOrchestratorService creates a new orchestrator service implementation
func NewOrchestratorService(
	aiDecisionEngine AIDecisionEngineInterface,
	graphExplorer GraphExplorerInterface,
	aiExecutionEngine AIExecutionEngineInterface,
	resultSynthesizer executionDomain.ResultSynthesizer,
	repository planningDomain.ExecutionPlanRepository,
	logger logging.Logger,
) *OrchestratorService {
	return &OrchestratorService{
		aiDecisionEngine:  aiDecisionEngine,
		graphExplorer:     graphExplorer,
		aiExecutionEngine: aiExecutionEngine,
		resultSynthesizer: resultSynthesizer,
		repository:        repository,
		logger:            logger,
	}
}

// OrchestratorRequest represents a user request to the orchestrator
type OrchestratorRequest struct {
	UserInput string `json:"user_input"`
	UserID    string `json:"user_id"`
	SessionID string `json:"session_id,omitempty"`
	MessageID string `json:"message_id,omitempty"` // ID of the user message that triggered this request
}

// OrchestratorResult represents the orchestrator's response
type OrchestratorResult struct {
	Message         string                   `json:"message"`
	Decision        *planningDomain.Decision `json:"decision"`
	Analysis        *planningDomain.Analysis `json:"analysis"`
	ExecutionPlanID string                   `json:"execution_plan_id,omitempty"`
	Success         bool                     `json:"success"`
	Error           string                   `json:"error,omitempty"`
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
	analysis, err := ors.aiDecisionEngine.ExploreAndAnalyze(ctx, request.UserInput, request.UserID, agentContext, request.MessageID)
	if err != nil {
		return &OrchestratorResult{
			Success: false,
			Error:   fmt.Sprintf("Failed to analyze request: %v", err),
		}, nil
	}

	decision, err := ors.aiDecisionEngine.MakeDecision(ctx, request.UserInput, request.UserID, analysis, request.MessageID)
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
	if decision.Type == planningDomain.DecisionTypeClarify {
		ors.logger.Info("🤔 Decision type: Clarify")
		result.Message = decision.ClarificationQuestion
	} else if decision.Type == planningDomain.DecisionTypeExecute {
		ors.logger.Info("🚀 Decision type: Execute", "requiredAgents", len(analysis.RequiredAgents))

		// Check if this is a meta-query that should be handled with AI orchestrator knowledge
		if ors.isOrchestratorMetaQuery(request.UserInput) {
			ors.logger.Info("🏛️ Meta-query detected, using AI to provide intelligent system insights")
			// Use AI conversation engine with orchestrator context for dynamic, intelligent responses
			result.Message = ors.handleMetaQuery(ctx, request.UserInput, agentContext)
		} else if len(analysis.RequiredAgents) > 0 {
			// AI-native execution: Use dedicated execution engine for agent coordination
			ors.logger.Info("🚀 Using AI execution engine with agents", "agents", analysis.RequiredAgents)

			// For now, use ExecutionPlanID as the plan text (backward compatibility)
			// TODO: In future iterations, retrieve structured plan and convert to execution steps
			executionPlan := decision.ExecutionPlanID
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
			result.Message = decision.ExecutionPlanID
		}
	} else {
		ors.logger.Warn("❓ Unknown decision type", "type", decision.Type)
	}

	ors.logger.Info("✅ Final result", "success", result.Success, "message", result.Message, "error", result.Error)

	// 4. Learning service removed for now (following YAGNI principles)
	// err = ors.learningService.StoreInsights(ctx, request.UserInput, analysis, decision)
	// if err != nil {
	//	ors.logger.Warn("Failed to store learning insights", "error", err)
	// }

	return result, nil
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
