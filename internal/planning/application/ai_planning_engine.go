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
// Uses unified ExecutionPlan entity (consolidates former PlanningResult + ExecutionPlan)
type AIPlanningEngine struct {
	aiProvider        aiDomain.AIProvider
	responseParser    *domain.ResponseParser
	executionPlanRepo domain.ExecutionPlanRepository // Unified repository
}

// NewAIPlanningEngine creates a new AI planning engine
func NewAIPlanningEngine(aiProvider aiDomain.AIProvider) *AIPlanningEngine {
	return &AIPlanningEngine{
		aiProvider:     aiProvider,
		responseParser: domain.NewResponseParser(),
	}
}

// NewAIPlanningEngineWithRepository creates a new AI planning engine with unified repository
func NewAIPlanningEngineWithRepository(aiProvider aiDomain.AIProvider, executionPlanRepo domain.ExecutionPlanRepository) *AIPlanningEngine {
	return &AIPlanningEngine{
		aiProvider:        aiProvider,
		responseParser:    domain.NewResponseParser(),
		executionPlanRepo: executionPlanRepo,
	}
}

// CreateExecutionPlan analyzes user request and creates unified execution plan
// Returns unified ExecutionPlan (consolidates former PlanningResult + ExecutionPlan creation)
// Supports optional conversation history for context-aware planning (variadic parameter for backward compatibility)
func (e *AIPlanningEngine) CreateExecutionPlan(ctx context.Context, userInput, userID, agentContext, requestID string, conversationHistory ...[]*aiDomain.AIConversationMessage) (*domain.ExecutionPlan, error) {
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

CAPABILITY GAP ANALYSIS:
Before choosing EXECUTE, perform this critical analysis:
1. Parse user request to identify ALL required capabilities (e.g., "word count", "translation", "deployment")
2. Check each available agent's capabilities against required capabilities
3. If ANY required capability is missing from available agents, choose CLARIFY instead of EXECUTE
4. Example: If user wants "count words and translate", but only text-processor (word count) is available and no translation agent exists, this is a capability gap - use CLARIFY

PLANNING_TYPES:
- EXECUTE: Create execution plan using available agents (ONLY when ALL required capabilities are available)
- CLARIFY: Ask for clarification when request is unclear, confidence < 80%, OR when required capabilities are missing from available agents

UNIFIED_ARCHITECTURE_RULES:
- Parse the JSON to find agents with matching capabilities for the user request
- Use exact agent "id" from the JSON (like "exact-agent-id-from-json")
- CRITICAL: Use CLARIFY when required capabilities are missing from available agents
- NO workaround execution plans - if capability is missing, clarify with user first
- Use descriptive clarification that explains what's available vs what's needed

Respond in this EXACT format:

PLANNING_RESULT:
Intent: [clear intent]
Category: [domain area]
Confidence: [0-100]
Available_Agents: [list agent names from JSON]
Required_Agents: [list specific agent IDs from JSON that match the request capabilities]
Planning_Type: [EXECUTE|CLARIFY]
Reasoning: [detailed reasoning for planning decisions - include capability gap analysis]

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
[contextual question explaining available capabilities vs required capabilities, with specific options]`

	userPrompt := fmt.Sprintf(`User ID: %s
Request: %s

Create a comprehensive execution plan for this request.`, userID, userInput)

	var response string
	var err error

	// Use conversation-aware AI if conversation history is provided
	if len(conversationHistory) > 0 && len(conversationHistory[0]) > 0 {
		// Build conversation with system prompt and user context
		conversation := []*aiDomain.AIConversationMessage{
			aiDomain.NewAIConversationMessage("system", systemPrompt),
		}

		// Add existing conversation history
		conversation = append(conversation, conversationHistory[0]...)

		// Add current user request as final message
		conversation = append(conversation, aiDomain.NewAIConversationMessage("user", userPrompt))

		// Call AI with full conversation context
		response, err = e.aiProvider.CallAIWithConversation(ctx, conversation)
	} else {
		// Fallback to standard single-turn AI call for backward compatibility
		response, err = e.aiProvider.CallAI(ctx, systemPrompt, userPrompt)
	}

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

	// Create unified execution plan based on planning type
	var unifiedPlan *domain.ExecutionPlan
	switch planningType {
	case "EXECUTE":
		// Extract execution plan details
		executionPlanJSON := e.responseParser.ExtractSection(response, "EXECUTION_PLAN:")
		if executionPlanJSON == "" {
			executionPlanJSON = e.responseParser.ExtractSection(response, "EXECUTION_PLAN_JSON:")
		}

		// Create unified execution plan with both planning intelligence and execution metadata
		unifiedPlan = domain.NewUnifiedExecutionPlan(
			domain.GenerateID(), // Generate new ID
			"AI Generated Plan",
			"Plan generated by AI planning engine",
			domain.ExecutionPlanPriorityMedium,
			requestID,
			intent,
			category,
			confidence,
			reasoning,
			availableAgentNames,
			requiredAgents,
			domain.PlanningTypeExecute,
		)

		// Parse and add execution steps if JSON provided
		if executionPlanJSON != "" {
			steps, err := e.parseExecutionPlanJSON(executionPlanJSON)
			if err != nil {
				return nil, fmt.Errorf("failed to parse execution plan JSON: %w", err)
			}

			for _, step := range steps {
				if err := unifiedPlan.AddStep(step); err != nil {
					return nil, fmt.Errorf("failed to add step to unified plan: %w", err)
				}
			}
		}

		// Mark planning as completed
		if err := unifiedPlan.CompletePlanning(); err != nil {
			return nil, fmt.Errorf("failed to complete planning phase: %w", err)
		}

	case "CLARIFY":
		clarificationQuestion := e.responseParser.ExtractSection(response, "CLARIFICATION:")

		// Create unified plan for clarification (will have empty steps until clarified)
		unifiedPlan = domain.NewUnifiedExecutionPlan(
			domain.GenerateID(), // Generate new ID
			"Clarification Needed",
			clarificationQuestion,
			domain.ExecutionPlanPriorityMedium,
			requestID,
			intent,
			category,
			confidence,
			reasoning,
			availableAgentNames,
			requiredAgents,
			domain.PlanningTypeClarify,
		)

		// Mark planning as completed (even for clarification)
		if err := unifiedPlan.CompletePlanning(); err != nil {
			return nil, fmt.Errorf("failed to complete planning phase: %w", err)
		}

	default:
		return nil, fmt.Errorf("unified architecture: unknown planning type '%s' - only EXECUTE and CLARIFY are supported", planningType)
	}

	// Store unified plan in repository if available
	if e.executionPlanRepo != nil {
		if err := e.executionPlanRepo.Create(ctx, unifiedPlan); err != nil {
			return nil, fmt.Errorf("failed to store unified execution plan: %w", err)
		}

		// Link to request (consolidated operation)
		if err := e.executionPlanRepo.LinkToRequest(ctx, unifiedPlan.ID, requestID); err != nil {
			return nil, fmt.Errorf("failed to link execution plan to request: %w", err)
		}
	}

	return unifiedPlan, nil
}

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
// NOTE: For execution plans, this is now a no-op as Conversation domain owns the relationship
func (e *AIPlanningEngine) LinkPlanningResultToConversation(ctx context.Context, planningResultID, conversationID string) error {
	if e.executionPlanRepo == nil {
		return fmt.Errorf("execution plan repository not available")
	}

	// Delegate to repository - for execution plans this will be a no-op to avoid duplicate relationships
	// Conversation domain handles conversation->execution_plan via LinkExecutionPlan
	return e.executionPlanRepo.LinkToConversation(ctx, planningResultID, conversationID)
}
