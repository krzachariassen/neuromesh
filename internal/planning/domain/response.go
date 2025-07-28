package domain

import (
	"fmt"
	"strings"
)

// ResponseParser handles parsing of AI responses into structured data
type ResponseParser struct{}

// NewResponseParser creates a new response parser
func NewResponseParser() *ResponseParser {
	return &ResponseParser{}
}

// ExtractSection extracts a specific section from AI response text
func (r *ResponseParser) ExtractSection(text, marker string) string {
	parts := strings.Split(text, marker)
	if len(parts) < 2 {
		return ""
	}

	section := parts[1]
	// Find the end of this section (next marker or end of text)
	nextMarkers := []string{
		"DECISION:", "CONFIDENCE:", "REASONING:", "CLARIFICATION:", "EXECUTION_PLAN:", "AGENT_COORDINATION:",
		"Intent:", "Category:", "Required_Agents:", "Reasoning:", "Confidence:",
		"Planning_Type:", "Available_Agents:", "EXECUTION_PLAN:", "DIRECT_RESPONSE:",
		"PLANNING_RESULT:", "EXECUTION_PLAN_JSON:",
	}
	minIndex := len(section)

	for _, nextMarker := range nextMarkers {
		if nextMarker != marker { // Don't end on the same marker we're looking for
			if idx := strings.Index(section, nextMarker); idx >= 0 && idx < minIndex {
				minIndex = idx
			}
		}
	}

	if minIndex < len(section) {
		section = section[:minIndex]
	}

	return strings.TrimSpace(section)
}

// ParseConfidence extracts confidence percentage from text
func (r *ResponseParser) ParseConfidence(confidenceStr string) int {
	// Simple extraction - look for numbers
	for i, char := range confidenceStr {
		if char >= '0' && char <= '9' {
			end := i
			for end < len(confidenceStr) && confidenceStr[end] >= '0' && confidenceStr[end] <= '9' {
				end++
			}
			if val := confidenceStr[i:end]; val != "" {
				var num int
				if n, err := fmt.Sscanf(val, "%d", &num); n == 1 && err == nil {
					return num
				}
			}
			break
		}
	}
	return 0
}

// ExtractIntent extracts and normalizes intent from analysis
func (r *ResponseParser) ExtractIntent(analysis string) string {
	intent := r.ExtractSection(analysis, "Intent:")
	if intent == "" {
		return "general_assistance"
	}
	return strings.ToLower(strings.ReplaceAll(intent, " ", "_"))
}

// ExtractCategory extracts and normalizes category from analysis
func (r *ResponseParser) ExtractCategory(analysis string) string {
	category := r.ExtractSection(analysis, "Category:")
	if category == "" {
		return "general"
	}
	return strings.ToLower(strings.ReplaceAll(category, " ", "_"))
}

// ExtractRequiredAgents parses required agents from analysis
func (r *ResponseParser) ExtractRequiredAgents(analysis string) []string {
	// Try both formats - with underscore and with space
	agentsStr := r.ExtractSection(analysis, "Required_Agents:")
	if agentsStr == "" {
		agentsStr = r.ExtractSection(analysis, "Required Agents:")
	}
	if agentsStr == "" {
		return []string{}
	}

	// Handle different formats:
	// 1. Comma-separated: "agent1, agent2, agent3"
	// 2. Single agent: "deployment-agent"
	// 3. Bracketed: "[agent1, agent2]" or "[agent1]"
	// 4. Empty or "none"

	agentsStr = strings.TrimSpace(agentsStr)
	if agentsStr == "" || agentsStr == "none" || agentsStr == "None" {
		return []string{}
	}

	// Remove brackets if present (AI might format as "[agent1, agent2]")
	agentsStr = strings.Trim(agentsStr, "[]")
	agentsStr = strings.TrimSpace(agentsStr)

	if agentsStr == "" {
		return []string{}
	}

	// Parse comma-separated agent names
	var agents []string
	if strings.Contains(agentsStr, ",") {
		// Multiple agents separated by commas
		agentParts := strings.Split(agentsStr, ",")
		for _, agent := range agentParts {
			agent = strings.TrimSpace(agent)
			if agent != "" && agent != "none" && agent != "None" {
				agents = append(agents, agent)
			}
		}
	} else {
		// Single agent
		if agentsStr != "none" && agentsStr != "None" {
			agents = append(agents, agentsStr)
		}
	}

	return agents
}

// ExtractPlanningType parses planning type from new planning format
func (r *ResponseParser) ExtractPlanningType(analysis string) string {
	planningType := r.ExtractSection(analysis, "Planning_Type:")
	if planningType == "" {
		planningType = r.ExtractSection(analysis, "PLANNING_TYPE:")
	}

	// Normalize to standard values
	planningType = strings.ToUpper(strings.TrimSpace(planningType))
	switch planningType {
	case "EXECUTE", "EXECUTION":
		return "EXECUTE"
	case "CLARIFY", "CLARIFICATION":
		return "CLARIFY"
	case "RESPOND_DIRECTLY", "DIRECT_RESPONSE", "RESPOND":
		return "RESPOND_DIRECTLY"
	default:
		return planningType
	}
}
