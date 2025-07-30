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
	CreateExecutionPlan(ctx context.Context, userInput, userID, agentContext, requestID string) (*planningDomain.ExecutionPlan, error)
	LinkPlanningResultToConversation(ctx context.Context, planningResultID, conversationID string) error
}

// GraphExplorerInterface defines the interface for graph exploration
type GraphExplorerInterface interface {
	GetAgentContext(ctx context.Context) (string, error)
}

// AIExecutionEngineInterface defines the interface for AI-native execution orchestration
type AIExecutionEngineInterface interface {
	ExecuteWithAgents(ctx context.Context, executionPlan, userInput, userID, agentContext, planID string) (string, error)
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
	Message         string                        `json:"message"`
	ExecutionPlan   *planningDomain.ExecutionPlan `json:"execution_plan,omitempty"` // Unified execution plan
	ExecutionPlanID string                        `json:"execution_plan_id,omitempty"`
	Status          string                        `json:"status,omitempty"` // NEW: Execution status for pure orchestration
	Success         bool                          `json:"success"`
	Error           string                        `json:"error,omitempty"`
}

// ProcessUserRequest is the main entry point that replaces the old ProcessRequest()
// This follows the clean architecture pattern with proper domain boundaries
// PHASE 3: Pure orchestration - only orchestrates, never executes
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
		ExecutionPlan:   planningResult,
		ExecutionPlanID: planningResult.ID,
		Success:         true,
	}

	// 3. Handle planning result based on type - PURE ORCHESTRATION
	if planningResult.Type == planningDomain.PlanningTypeClarify {
		ors.logger.Info("🤔 Planning type: Clarify")
		result.Message = planningResult.Reasoning // Use reasoning instead of ClarificationQuestion
	} else if planningResult.Type == planningDomain.PlanningTypeExecute {
		ors.logger.Info("🚀 Planning type: Execute")
		result.ExecutionPlanID = planningResult.ID

		// PURE ORCHESTRATION: We no longer convert or execute directly
		// Instead, we just prepare the execution plan and pass it along
		// AI-native execution orchestration - completely asynchronous
		go func() {
			backgroundCtx := context.Background()
			ors.logger.Info("🤖 Starting AI-native execution", "planID", planningResult.ID)
			// Convert ExecutionPlan to JSON string for the execution engine
			executionPlanJSON := ors.convertExecutionPlanToJSON(planningResult)
			finalResult, err := ors.aiExecutionEngine.ExecuteWithAgents(backgroundCtx, executionPlanJSON, request.UserInput, request.UserID, agentContext, planningResult.ID)
			if err != nil {
				ors.logger.Error("❌ AI-native execution failed", err, "planID", planningResult.ID)
			} else {
				ors.logger.Info("✅ AI-native execution completed", "planID", planningResult.ID, "result", finalResult)
			}
		}()

		ors.logger.Info("✅ AI-native async execution started", "planID", planningResult.ID)
		result.Status = "executing"
		// NO immediate message - pure orchestration returns plan ID only
	} else {
		ors.logger.Warn("❓ Unknown planning type", "type", planningResult.Type)
	}

	ors.logger.Info("✅ Pure orchestration result", "success", result.Success, "planID", result.ExecutionPlanID, "status", result.Status)

	// 4. Cross-domain coordination: Link execution plans to conversations
	if planningResult.Type == planningDomain.PlanningTypeExecute && planningResult.ID != "" && request.ConversationID != "" {
		if err := ors.coordinateCrossDomainRelationships(ctx, request.ConversationID, planningResult.ID); err != nil {
			ors.logger.Warn("Failed to coordinate cross-domain relationships", "error", err, "conversationID", request.ConversationID, "planID", planningResult.ID)
			// Don't fail the entire request for relationship linking issues
		} else {
			ors.logger.Info("✅ Cross-domain relationship created", "conversationID", request.ConversationID, "planID", planningResult.ID)
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

// convertExecutionPlanToJSON converts an execution plan to JSON string for the execution engine
func (ors *OrchestratorService) convertExecutionPlanToJSON(executionPlan *planningDomain.ExecutionPlan) string {
	if executionPlan == nil {
		return "{}"
	}

	// Create a simplified execution plan format for the execution engine
	executionPlanStr := fmt.Sprintf(`{
	"id": "%s",
	"intent": "%s",
	"category": "%s",
	"required_agents": %s,
	"agent_gap": %s,
	"reasoning": "%s",
	"steps": %s
}`,
		executionPlan.ID,
		executionPlan.Intent,
		executionPlan.Category,
		ors.formatAgentList(executionPlan.RequiredAgents),
		ors.formatAgentList(executionPlan.AgentGap),
		executionPlan.Reasoning,
		ors.formatSteps(executionPlan.Steps))

	return executionPlanStr
}

// formatAgentList formats a slice of agents as JSON array string
func (ors *OrchestratorService) formatAgentList(agents []string) string {
	if len(agents) == 0 {
		return "[]"
	}

	formatted := "["
	for i, agent := range agents {
		if i > 0 {
			formatted += ", "
		}
		formatted += fmt.Sprintf(`"%s"`, agent)
	}
	formatted += "]"
	return formatted
}

// formatSteps formats execution steps as JSON array string
func (ors *OrchestratorService) formatSteps(steps []*planningDomain.ExecutionStep) string {
	if len(steps) == 0 {
		return "[]"
	}

	formatted := "["
	for i, step := range steps {
		if i > 0 {
			formatted += ", "
		}
		formatted += fmt.Sprintf(`{
			"id": "%s",
			"description": "%s",
			"agent": "%s"
		}`, step.ID, step.Description, step.AssignedAgent)
	}
	formatted += "]"
	return formatted
}

// convertPlanningResultToExecutionPlan converts a planning result to execution plan format (legacy support)
func (ors *OrchestratorService) convertPlanningResultToExecutionPlan(planningResult *planningDomain.ExecutionPlan) string {
	if planningResult == nil {
		return ""
	}

	// This method is now simplified since we work directly with ExecutionPlan
	executionPlan := fmt.Sprintf(`Execution Plan ID: %s
Intent: %s
Category: %s
Required Agents: %v
Agent Gap: %v
Reasoning: %s`,
		planningResult.ID,
		planningResult.Intent,
		planningResult.Category,
		planningResult.RequiredAgents,
		planningResult.AgentGap,
		planningResult.Reasoning)

	return executionPlan
}
