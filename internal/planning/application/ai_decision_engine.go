package application

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	aiDomain "neuromesh/internal/ai/domain"
	orchestratorDomain "neuromesh/internal/orchestrator/domain"
	"neuromesh/internal/planning/domain"
)

// AIDecisionEngine handles AI-powered decision making
type AIDecisionEngine struct {
	aiProvider        aiDomain.AIProvider
	responseParser    *domain.ResponseParser
	executionPlanRepo domain.ExecutionPlanRepository
}

// NewAIDecisionEngine creates a new AI decision engine
func NewAIDecisionEngine(aiProvider aiDomain.AIProvider) *AIDecisionEngine {
	return &AIDecisionEngine{
		aiProvider:     aiProvider,
		responseParser: domain.NewResponseParser(),
	}
}

// NewAIDecisionEngineWithRepository creates a new AI decision engine with execution plan repository
func NewAIDecisionEngineWithRepository(aiProvider aiDomain.AIProvider, executionPlanRepo domain.ExecutionPlanRepository) *AIDecisionEngine {
	return &AIDecisionEngine{
		aiProvider:        aiProvider,
		responseParser:    domain.NewResponseParser(),
		executionPlanRepo: executionPlanRepo,
	}
}

// ExploreAndAnalyze analyzes user request with agent context and returns structured analysis
func (e *AIDecisionEngine) ExploreAndAnalyze(ctx context.Context, userInput, userID, agentContext, requestID string) (*domain.Analysis, error) {
	systemPrompt := `You are an AI orchestrator. You have access to the following agents and their capabilities:

AVAILABLE_AGENTS:
` + agentContext + `

Analyze the user request and determine:
- Intent: What does the user want to accomplish?
- Category: What domain/area (deployment, security, monitoring, etc.)?
- Confidence: How confident are you in understanding the request?
- Required_Agents: Which agents (if any) would be needed to fulfill this request?

Respond in this format:
ANALYSIS:
Intent: [clear intent]
Category: [domain area]
Confidence: [0-100 percent]
Required_Agents: [list specific agents needed]
Reasoning: [why this analysis]`

	userPrompt := fmt.Sprintf(`User ID: %s
Request: %s

Analyze this request based on available agents.`, userID, userInput)

	response, err := e.aiProvider.CallAI(ctx, systemPrompt, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("AI call failed: %w", err)
	}

	// Parse the response into structured analysis
	intent := e.responseParser.ExtractIntent(response)
	category := e.responseParser.ExtractCategory(response)
	confidenceStr := e.responseParser.ExtractSection(response, "Confidence:")
	confidence := e.responseParser.ParseConfidence(confidenceStr)
	requiredAgents := e.responseParser.ExtractRequiredAgents(response)
	reasoning := e.responseParser.ExtractSection(response, "Reasoning:")

	// Use the provided requestID (which comes from conversation messageID)
	return domain.NewAnalysis(requestID, intent, category, confidence, requiredAgents, reasoning), nil
}

// MakeDecision determines whether to clarify or execute based on analysis
// Returns planning decisions only - orchestrator handles execution coordination
func (e *AIDecisionEngine) MakeDecision(ctx context.Context, userInput, userID string, analysis *domain.Analysis, requestID string) (*orchestratorDomain.Decision, error) {
	systemPrompt := `You are an AI orchestrator that decides whether to ask for clarification or execute a request.

Based on the provided analysis, you must:

1. ASSESS if you need clarification (confidence < 80 percent OR complex multi-step request)
2. IF clarification needed: Generate a helpful clarification question
3. IF ready to execute: Provide comprehensive execution plan with agent coordination

Your analysis includes graph context with available agents and capabilities. When generating execution plans, you MUST:
- Reference specific agents by name that were found in the graph exploration
- Use EXACT agent names from the analysis
- Create realistic agent coordination plans

Respond in this EXACT format:

DECISION: [CLARIFY|EXECUTE]
CONFIDENCE: [0-100]
REASONING: [why this decision]

[If CLARIFY]:
CLARIFICATION: [specific question to ask]

[If EXECUTE]:
EXECUTION_PLAN_JSON:
{
  "steps": [
    {
      "step_number": 1,
      "agent_name": "exact-agent-name-from-analysis",
      "action_description": "specific action description",
      "step_name": "brief step name"
    },
    {
      "step_number": 2,
      "agent_name": "exact-agent-name-from-analysis", 
      "action_description": "specific action description",
      "step_name": "brief step name"
    }
  ]
}

AGENT_COORDINATION:
- Primary Agent: [specific agent name from analysis and why]
- Supporting Agents: [list specific agent names and roles]
- Workflow Dependencies: [any sequencing needed]`

	analysisText := fmt.Sprintf(`Intent: %s
Category: %s
Confidence: %d
Required_Agents: %s
Reasoning: %s`, analysis.Intent, analysis.Category, analysis.Confidence, strings.Join(analysis.RequiredAgents, ", "), analysis.Reasoning)

	userPrompt := fmt.Sprintf(`User ID: %s
Original Request: %s

ANALYSIS:
%s

Based on this analysis, decide whether to clarify or execute.`, userID, userInput, analysisText)

	response, err := e.aiProvider.CallAI(ctx, systemPrompt, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("AI call failed: %w", err)
	}

	// Parse the decision
	if strings.Contains(response, "DECISION: CLARIFY") {
		clarificationQuestion := e.responseParser.ExtractSection(response, "CLARIFICATION:")
		reasoning := e.responseParser.ExtractSection(response, "REASONING:")
		return orchestratorDomain.NewClarifyDecision(requestID, analysis.ID, clarificationQuestion, reasoning), nil
	}

	// For execution decisions, create and persist structured ExecutionPlan
	executionPlanJSON := e.responseParser.ExtractSection(response, "EXECUTION_PLAN_JSON:")
	agentCoordination := e.responseParser.ExtractSection(response, "AGENT_COORDINATION:")
	reasoning := e.responseParser.ExtractSection(response, "REASONING:")

	// If we have an ExecutionPlanRepository, create and persist structured plan
	var executionPlanID string
	if e.executionPlanRepo != nil {
		// Parse the JSON execution plan into structured steps
		steps, err := e.parseExecutionPlanJSON(executionPlanJSON)
		if err != nil {
			return nil, fmt.Errorf("failed to parse execution plan JSON: %w", err)
		}

		// Create ExecutionPlan with steps
		plan := domain.NewExecutionPlan("AI Generated Plan", "Plan generated by AI decision engine", domain.ExecutionPlanPriorityMedium)
		for _, step := range steps {
			if err := plan.AddStep(step); err != nil {
				return nil, fmt.Errorf("failed to add step to plan: %w", err)
			}
		}

		// Persist the plan to the graph
		if err := e.executionPlanRepo.Create(ctx, plan); err != nil {
			return nil, fmt.Errorf("failed to persist execution plan: %w", err)
		}

		// Link the plan to the analysis
		if err := e.executionPlanRepo.LinkToAnalysis(ctx, analysis.ID, plan.ID); err != nil {
			return nil, fmt.Errorf("failed to link execution plan to analysis: %w", err)
		}

		executionPlanID = plan.ID
	} else {
		// Fallback: use the execution plan JSON as ID (backward compatibility)
		executionPlanID = executionPlanJSON
	}

	// Return a planning recommendation that execution should happen
	// Note: This creates a unified decision for now, but orchestrator coordinates domains
	return orchestratorDomain.NewExecuteDecision(requestID, analysis.ID, executionPlanID, agentCoordination, reasoning), nil
}

// parseExecutionPlanJSON parses JSON execution plan into structured steps
func (e *AIDecisionEngine) parseExecutionPlanJSON(jsonStr string) ([]*domain.ExecutionStep, error) {
	// Clean up the JSON string
	jsonStr = strings.TrimSpace(jsonStr)
	if jsonStr == "" {
		return nil, fmt.Errorf("execution plan JSON is empty")
	}

	// Define the JSON structure we expect from the AI
	type StepJSON struct {
		StepNumber        int    `json:"step_number"`
		AgentName         string `json:"agent_name"`
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
		if stepJSON.AgentName == "" {
			return nil, fmt.Errorf("step %d: agent_name cannot be empty", stepJSON.StepNumber)
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

		// Create ExecutionStep
		step := domain.NewExecutionStep(stepName, stepJSON.ActionDescription, stepJSON.AgentName)
		step.StepNumber = stepJSON.StepNumber
		steps = append(steps, step)
	}

	return steps, nil
}
