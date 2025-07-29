package application

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	aiDomain "neuromesh/internal/ai/domain"
	"neuromesh/internal/planning/domain"
)

// AIPlanningEngine handles AI-powered planning for user requests
// This replaces the confusing "AIDecisionEngine" with proper planning terminology
type AIPlanningEngine struct {
	aiProvider         aiDomain.AIProvider
	responseParser     *domain.ResponseParser
	executionPlanRepo  domain.ExecutionPlanRepository
	planningResultRepo domain.PlanningResultRepository
}

// NewAIPlanningEngine creates a new AI planning engine
func NewAIPlanningEngine(aiProvider aiDomain.AIProvider) *AIPlanningEngine {
	return &AIPlanningEngine{
		aiProvider:     aiProvider,
		responseParser: domain.NewResponseParser(),
	}
}

// NewAIPlanningEngineWithRepository creates a new AI planning engine with execution plan repository
func NewAIPlanningEngineWithRepository(aiProvider aiDomain.AIProvider, executionPlanRepo domain.ExecutionPlanRepository) *AIPlanningEngine {
	return &AIPlanningEngine{
		aiProvider:        aiProvider,
		responseParser:    domain.NewResponseParser(),
		executionPlanRepo: executionPlanRepo,
	}
}

// NewAIPlanningEngineWithRepositories creates a new AI planning engine with both repositories
func NewAIPlanningEngineWithRepositories(aiProvider aiDomain.AIProvider, executionPlanRepo domain.ExecutionPlanRepository, planningResultRepo domain.PlanningResultRepository) *AIPlanningEngine {
	return &AIPlanningEngine{
		aiProvider:         aiProvider,
		responseParser:     domain.NewResponseParser(),
		executionPlanRepo:  executionPlanRepo,
		planningResultRepo: planningResultRepo,
	}
}

// CreateExecutionPlan analyzes user request and creates comprehensive execution plan
// This replaces the two-step ExploreAndAnalyze -> MakeDecision with unified planning
func (e *AIPlanningEngine) CreateExecutionPlan(ctx context.Context, userInput, userID, agentContext, requestID string) (*domain.PlanningResult, error) {
	// Parse available agents from agent context - get names for display
	availableAgentNames, _ := e.parseAvailableAgentsWithIDs(agentContext)

	systemPrompt := `You are an AI planning orchestrator. Analyze the user request and create a comprehensive execution plan.

AVAILABLE_AGENTS (JSON format):
` + agentContext + `

Your planning must determine:
1. Intent: What does the user want to accomplish?
2. Category: What domain/area (deployment, security, monitoring, guidance, etc.)?
3. Confidence: How confident are you in understanding the request (0-100)?
4. Required_Agents: Which specific agents are needed to fulfill this request?
5. Planning_Type: How should this request be handled?

The available agents are provided in JSON format. Parse the JSON to extract exact agent "id" fields and "capabilities".

PLANNING_TYPES:
- EXECUTE: Create execution plan using available agents (PREFERRED - use available agents for requests)
- CLARIFY: Ask for clarification when request is unclear or confidence < 80%

UNIFIED_ARCHITECTURE_RULES:
- ALL requests should use EXECUTE with execution plans
- Parse the JSON to find agents with matching capabilities for the user request
- Use exact agent "id" from the JSON (like ""exact-agent-id-from-json")
- Only use CLARIFY when the request is genuinely ambiguous or unclear
- NO direct responses - everything goes through agent execution for consistency

Respond in this EXACT format:

PLANNING_RESULT:
Intent: [clear intent]
Category: [domain area]
Confidence: [0-100]
Available_Agents: [list agent names from JSON]
Required_Agents: [list specific agent IDs from JSON that match the request capabilities]
Planning_Type: [EXECUTE|CLARIFY]
Reasoning: [detailed reasoning for planning decisions - this is internal analysis, NOT user-facing content]

[If EXECUTE]:
EXECUTION_PLAN:
{
  "steps": [
    {
      "step_number": 1,
      "agent_id": "exact-agent-id-from-json",
      "action_description": "specific action description",
      "step_name": "brief step name"
    }
  ]
}

[If CLARIFY]:
CLARIFICATION:
[question to ask user for clarification]`

	userPrompt := fmt.Sprintf(`User ID: %s
Request: %s

Create a comprehensive execution plan for this request.`, userID, userInput)

	response, err := e.aiProvider.CallAI(ctx, systemPrompt, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("AI planning call failed: %w", err)
	}

	// Parse the planning response
	intent := e.responseParser.ExtractIntent(response)
	category := e.responseParser.ExtractCategory(response)
	confidenceStr := e.responseParser.ExtractSection(response, "Confidence:")
	confidence := e.responseParser.ParseConfidence(confidenceStr)
	requiredAgents := e.responseParser.ExtractRequiredAgents(response)
	reasoning := e.responseParser.ExtractSection(response, "Reasoning:")
	planningType := e.responseParser.ExtractPlanningType(response)

	// Create planning result based on the type
	var planningResult *domain.PlanningResult
	switch planningType {
	case "EXECUTE":
		// Extract execution plan details - try both EXECUTION_PLAN and EXECUTION_PLAN_JSON
		executionPlanJSON := e.responseParser.ExtractSection(response, "EXECUTION_PLAN:")
		if executionPlanJSON == "" {
			executionPlanJSON = e.responseParser.ExtractSection(response, "EXECUTION_PLAN_JSON:")
		}

		// Parse execution plan into structured steps if we have a repository
		var executionPlanID string
		if e.executionPlanRepo != nil && executionPlanJSON != "" {
			steps, err := e.parseExecutionPlanJSON(executionPlanJSON)
			if err != nil {
				return nil, fmt.Errorf("failed to parse execution plan JSON: %w", err)
			}

			// Create ExecutionPlan with steps
			plan := domain.NewExecutionPlan("AI Generated Plan", "Plan generated by planning engine", domain.ExecutionPlanPriorityMedium)
			for _, step := range steps {
				if err := plan.AddStep(step); err != nil {
					return nil, fmt.Errorf("failed to add step to plan: %w", err)
				}
			}

			// Persist the plan to the graph
			if err := e.executionPlanRepo.Create(ctx, plan); err != nil {
				return nil, fmt.Errorf("failed to store execution plan: %w", err)
			}
			executionPlanID = plan.ID
		}

		// Use agent names for display in available agents, but IDs for gap calculation
		planningResult = domain.NewExecutePlanningResult(
			requestID, intent, category, confidence, availableAgentNames,
			requiredAgents, executionPlanID, reasoning,
		)

	case "CLARIFY":
		clarificationQuestion := e.responseParser.ExtractSection(response, "CLARIFICATION:")
		planningResult = domain.NewClarificationPlanningResult(
			requestID, intent, category, confidence, clarificationQuestion, reasoning,
		)

	default:
		return nil, fmt.Errorf("unified architecture: unknown planning type '%s' - only EXECUTE and CLARIFY are supported", planningType)
	}

	// Store planning result in repository if available
	if e.planningResultRepo != nil {
		if err := e.planningResultRepo.Store(ctx, planningResult); err != nil {
			return nil, fmt.Errorf("failed to store planning result: %w", err)
		}

		// Link to request
		if err := e.planningResultRepo.LinkToRequest(ctx, planningResult.ID, requestID); err != nil {
			return nil, fmt.Errorf("failed to link planning result to request: %w", err)
		}

		// Link to execution plan if one was created
		if planningResult.ExecutionPlanID != "" {
			if err := e.planningResultRepo.LinkToExecutionPlan(ctx, planningResult.ID, planningResult.ExecutionPlanID); err != nil {
				return nil, fmt.Errorf("failed to link planning result to execution plan: %w", err)
			}
		}
	}

	return planningResult, nil
}

// parseExecutionPlanJSON parses JSON execution plan into structured steps
// parseExecutionPlanJSON parses JSON execution plan into structured steps
func (e *AIPlanningEngine) parseExecutionPlanJSON(jsonStr string) ([]*domain.ExecutionStep, error) {
	// Clean up the JSON string
	jsonStr = strings.TrimSpace(jsonStr)
	if jsonStr == "" {
		return nil, fmt.Errorf("execution plan JSON is empty")
	}

	// Define the JSON structure we expect from the AI
	type StepJSON struct {
		StepNumber        int    `json:"step_number"`
		AgentID           string `json:"agent_id"`
		ActionDescription string `json:"action_description"`
		StepName          string `json:"step_name"`
	}

	type ExecutionPlanJSON struct {
		Steps []StepJSON `json:"steps"`
	}

	// Parse the JSON
	var planJSON ExecutionPlanJSON
	if err := json.Unmarshal([]byte(jsonStr), &planJSON); err != nil {
		return nil, fmt.Errorf("failed to parse execution plan JSON: %w", err)
	}

	// Convert JSON steps to domain ExecutionStep objects
	var steps []*domain.ExecutionStep
	for _, stepJSON := range planJSON.Steps {
		// Validate required fields
		if stepJSON.AgentID == "" {
			return nil, fmt.Errorf("step %d: agent_id cannot be empty", stepJSON.StepNumber)
		}
		if stepJSON.ActionDescription == "" {
			return nil, fmt.Errorf("step %d: action_description cannot be empty", stepJSON.StepNumber)
		}

		// Use provided step name or generate from action description
		stepName := stepJSON.StepName
		if stepName == "" {
			descWords := strings.Fields(stepJSON.ActionDescription)
			if len(descWords) >= 3 {
				stepName = strings.Join(descWords[:3], " ")
			} else {
				stepName = stepJSON.ActionDescription
			}
		}

		// Create ExecutionStep - now correctly using AgentID
		step := domain.NewExecutionStep(stepName, stepJSON.ActionDescription, stepJSON.AgentID)
		step.StepNumber = stepJSON.StepNumber
		steps = append(steps, step)
	}

	return steps, nil
}

// parseAvailableAgents extracts available agent names from JSON agent context
func (e *AIPlanningEngine) parseAvailableAgents(agentContext string) []string {
	if agentContext == "" || agentContext == "No agents available" || agentContext == "No agents currently registered in the system" {
		return []string{}
	}

	// Try to parse as JSON first (new format)
	type AgentContext struct {
		ID           string `json:"id"`
		Name         string `json:"name"`
		Status       string `json:"status"`
		Capabilities []struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		} `json:"capabilities"`
	}

	type AgentContextResponse struct {
		AvailableAgents []AgentContext `json:"available_agents"`
	}

	var response AgentContextResponse
	if err := json.Unmarshal([]byte(agentContext), &response); err == nil {
		// Successfully parsed JSON format
		var agents []string
		for _, agent := range response.AvailableAgents {
			agents = append(agents, agent.Name) // Use name for planning result display
		}
		return agents
	}

	// Fallback to old text format parsing for backward compatibility
	lines := strings.Split(agentContext, "\n")
	var agents []string
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip header lines and empty lines
		if line == "" || strings.Contains(line, "Available Agents") {
			continue
		}

		// Handle different agent list formats:
		// Format 1: "- agent-name | Status: active"
		// Format 2: "agent-name: description"
		if strings.HasPrefix(line, "-") {
			// Remove the dash prefix and extract agent name
			line = strings.TrimPrefix(line, "-")
			line = strings.TrimSpace(line)

			// Extract agent name before the pipe or colon
			if idx := strings.Index(line, "|"); idx >= 0 {
				agentName := strings.TrimSpace(line[:idx])
				if agentName != "" {
					agents = append(agents, agentName)
				}
			} else if idx := strings.Index(line, ":"); idx >= 0 {
				agentName := strings.TrimSpace(line[:idx])
				if agentName != "" {
					agents = append(agents, agentName)
				}
			}
		} else if strings.Contains(line, ":") {
			// Format: "agent-name: description"
			parts := strings.Split(line, ":")
			if len(parts) > 0 {
				agentName := strings.TrimSpace(parts[0])
				if agentName != "" {
					agents = append(agents, agentName)
				}
			}
		}
	}

	return agents
}

// parseAvailableAgentsWithIDs extracts both agent names and IDs from JSON agent context
func (e *AIPlanningEngine) parseAvailableAgentsWithIDs(agentContext string) ([]string, []string) {
	if agentContext == "" || agentContext == "No agents available" || agentContext == "No agents currently registered in the system" {
		return []string{}, []string{}
	}

	// Try to parse as JSON first (new format)
	type AgentContext struct {
		ID           string `json:"id"`
		Name         string `json:"name"`
		Status       string `json:"status"`
		Capabilities []struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		} `json:"capabilities"`
	}

	type AgentContextResponse struct {
		AvailableAgents []AgentContext `json:"available_agents"`
	}

	var response AgentContextResponse
	if err := json.Unmarshal([]byte(agentContext), &response); err == nil {
		// Successfully parsed JSON format
		var agentNames []string
		var agentIDs []string
		for _, agent := range response.AvailableAgents {
			agentNames = append(agentNames, agent.Name) // Use name for planning result display
			agentIDs = append(agentIDs, agent.ID)       // Use ID for gap calculation
		}
		return agentNames, agentIDs
	}

	// Fallback to old text format parsing for backward compatibility
	// In this case, we can't distinguish between names and IDs, so return the same for both
	agents := e.parseAvailableAgents(agentContext)
	return agents, agents
}

// LinkPlanningResultToConversation links a planning result to a conversation
// This allows the orchestrator to coordinate cross-domain relationships
func (e *AIPlanningEngine) LinkPlanningResultToConversation(ctx context.Context, planningResultID, conversationID string) error {
	if e.planningResultRepo == nil {
		return fmt.Errorf("planning result repository not available")
	}

	return e.planningResultRepo.LinkToConversation(ctx, planningResultID, conversationID)
}
