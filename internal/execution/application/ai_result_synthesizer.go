package application

import (
	"context"
	"fmt"
	"strings"

	aiDomain "neuromesh/internal/ai/domain"
	executionDomain "neuromesh/internal/execution/domain"
	planningDomain "neuromesh/internal/planning/domain"
)

// AIResultSynthesizer implements ResultSynthesizer using AI to intelligently combine agent results
type AIResultSynthesizer struct {
	aiProvider aiDomain.AIProvider
	repository planningDomain.ExecutionPlanRepository
}

// NewAIResultSynthesizer creates a new AI-powered result synthesizer
func NewAIResultSynthesizer(aiProvider aiDomain.AIProvider, repository planningDomain.ExecutionPlanRepository) *AIResultSynthesizer {
	return &AIResultSynthesizer{
		aiProvider: aiProvider,
		repository: repository,
	}
}

// SynthesizeResults takes all agent results for an execution plan and creates a synthesized output
func (s *AIResultSynthesizer) SynthesizeResults(ctx context.Context, planID string) (string, error) {
	if planID == "" {
		return "", fmt.Errorf("planID cannot be empty")
	}

	// Get synthesis context with all agent results
	synthCtx, err := s.GetSynthesisContext(ctx, planID)
	if err != nil {
		return "", fmt.Errorf("failed to get synthesis context for plan %s: %w", planID, err)
	}

	// Validate we have some results to synthesize
	if len(synthCtx.AgentResults) == 0 {
		return "", fmt.Errorf("no agent results found for execution plan %s", planID)
	}

	// Build synthesis prompt with agent results
	systemPrompt := s.buildSynthesisSystemPrompt()
	userPrompt := s.buildSynthesisUserPrompt(synthCtx)

	// Use AI to synthesize results
	synthesizedResult, err := s.aiProvider.CallAI(ctx, systemPrompt, userPrompt)
	if err != nil {
		return "", fmt.Errorf("AI synthesis failed for plan %s: %w", planID, err)
	}

	// Validate the synthesized result is not empty
	if strings.TrimSpace(synthesizedResult) == "" {
		return "", fmt.Errorf("AI synthesis produced empty result for plan %s", planID)
	}

	return synthesizedResult, nil
}

// GetSynthesisContext retrieves and structures all data needed for synthesis
func (s *AIResultSynthesizer) GetSynthesisContext(ctx context.Context, planID string) (*executionDomain.SynthesisContext, error) {
	// Get all agent results for the execution plan
	agentResults, err := s.repository.GetAgentResultsByExecutionPlan(ctx, planID)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent results: %w", err)
	}

	// Create synthesis context
	synthCtx := executionDomain.NewSynthesisContext(planID, agentResults)

	// Add metadata if available
	synthCtx.AddMetadata("total_results", len(agentResults))
	synthCtx.AddMetadata("successful_results", len(synthCtx.GetSuccessfulResults()))

	return synthCtx, nil
}

// buildSynthesisSystemPrompt creates the system prompt for AI synthesis
func (s *AIResultSynthesizer) buildSynthesisSystemPrompt() string {
	return `You are an expert AI analyst specialized in synthesizing multi-agent execution results into comprehensive, actionable reports.

CORE RESPONSIBILITIES:
1. Analyze all agent execution results and extract key insights
2. Identify patterns, correlations, and relationships between agent outputs
3. Create a unified, well-structured response that intelligently combines all findings
4. Handle mixed success/failure scenarios professionally
5. Present information clearly and actionably for end users

SYNTHESIS GUIDELINES:
• Focus on actionable insights and critical findings
• Maintain complete accuracy - only synthesize what agents actually reported
• When agents fail, acknowledge limitations while highlighting successful results
• Structure responses logically with clear sections and flow
• Use professional, domain-appropriate language
• Quantify findings when specific metrics are available
• Provide context and implications for the synthesized results

OUTPUT REQUIREMENTS:
• Begin with an executive summary of key findings
• Present agent results in logical order (not chronological)
• Highlight the most important insights prominently
• Include specific data points and metrics when available
• End with actionable recommendations or next steps
• Maintain a tone appropriate for the business domain`
}

// buildSynthesisUserPrompt creates the user prompt with agent results
func (s *AIResultSynthesizer) buildSynthesisUserPrompt(synthCtx *executionDomain.SynthesisContext) string {
	var prompt strings.Builder

	prompt.WriteString("EXECUTION SYNTHESIS REQUEST\n\n")

	prompt.WriteString(fmt.Sprintf("Plan ID: %s\n", synthCtx.ExecutionPlanID))
	prompt.WriteString(fmt.Sprintf("Total Agent Results: %d\n", len(synthCtx.AgentResults)))
	prompt.WriteString(fmt.Sprintf("Successful Results: %d\n", len(synthCtx.GetSuccessfulResults())))
	prompt.WriteString(fmt.Sprintf("Failed Results: %d\n\n", len(synthCtx.AgentResults)-len(synthCtx.GetSuccessfulResults())))

	// Add metadata context if available
	if len(synthCtx.Metadata) > 0 {
		prompt.WriteString("EXECUTION CONTEXT:\n")
		for key, value := range synthCtx.Metadata {
			prompt.WriteString(fmt.Sprintf("• %s: %v\n", key, value))
		}
		prompt.WriteString("\n")
	}

	prompt.WriteString("AGENT EXECUTION RESULTS:\n\n")

	// Add each agent result with structured formatting
	for i, result := range synthCtx.AgentResults {
		prompt.WriteString(fmt.Sprintf("=== AGENT RESULT %d ===\n", i+1))
		prompt.WriteString(fmt.Sprintf("Agent ID: %s\n", result.AgentID))
		prompt.WriteString(fmt.Sprintf("Execution Status: %s\n", result.Status))
		prompt.WriteString(fmt.Sprintf("Timestamp: %s\n", result.Timestamp.Format("2006-01-02 15:04:05")))
		prompt.WriteString(fmt.Sprintf("\nResult Content:\n%s\n", result.Content))

		// Add structured metadata
		if len(result.Metadata) > 0 {
			prompt.WriteString("\nResult Metadata:\n")
			for key, value := range result.Metadata {
				prompt.WriteString(fmt.Sprintf("  • %s: %v\n", key, value))
			}
		}
		prompt.WriteString("\n")
	}

	prompt.WriteString("SYNTHESIS INSTRUCTIONS:\n")
	prompt.WriteString("Please analyze these agent execution results and provide a comprehensive synthesis that:\n")
	prompt.WriteString("1. Summarizes the key findings from all successful agents\n")
	prompt.WriteString("2. Identifies important patterns or correlations in the data\n")
	prompt.WriteString("3. Addresses any failed executions and their impact\n")
	prompt.WriteString("4. Provides actionable insights and recommendations\n")
	prompt.WriteString("5. Structures the response professionally for end-user consumption\n\n")
	prompt.WriteString("Focus on creating value from the collective agent outputs rather than simply concatenating individual results.")

	return prompt.String()
}
