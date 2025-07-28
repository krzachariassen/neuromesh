// REVISED AI-Native Execution Model
// Constrained by available agent registry, but still AI-driven discovery

package concepts

import (
	"context"
)

// AvailableAgent represents a registered agent in the platform
type AvailableAgent struct {
	ID           string
	Name         string
	Capabilities []string
	Status       string // active, inactive, busy
	Metadata     map[string]interface{}
}

// AgentCapabilityGap represents missing functionality
type AgentCapabilityGap struct {
	RequiredCapability string
	Context            string
	SuggestedAgents    []string // Agents that might partially help
	UserGuidance       string   // What to tell the user
}

// AIAgentDiscovery - AI explores what's possible with available agents
type AIAgentDiscovery struct {
	UserGoal        string
	AvailableAgents []*AvailableAgent
	SelectedAgents  []*AvailableAgent
	CapabilityGaps  []*AgentCapabilityGap
	ConfidenceLevel float64 // How confident AI is it can fulfill the request
}

// AIExplorationNode - AI discovers what agents can contribute
type AIExplorationNode struct {
	ID                   string
	Goal                 string                // What needs to be accomplished
	RequiredCapabilities []string              // What capabilities are needed
	MatchingAgents       []*AvailableAgent     // Agents that can help with this
	CapabilityGaps       []*AgentCapabilityGap // What's missing
	Dependencies         []string              // Other nodes this depends on
	SpawnedFrom          string                // Which discovery led to this node
}

// REVISED: AI-Native Agent Discovery Service
type AIAgentDiscoveryService interface {
	// AI analyzes user goal against available agent capabilities
	ExploreAgentCapabilities(ctx context.Context, userGoal string) (*AIAgentDiscovery, error)

	// AI discovers what additional capabilities might be needed as agents execute
	DiscoverAdditionalNeeds(ctx context.Context, nodeID string, agentFindings map[string]interface{}) (*AIExplorationNode, error)

	// AI determines if the goal is achievable with current agents
	EvaluateGoalFeasibility(ctx context.Context, discovery *AIAgentDiscovery) (bool, []string, error)

	// AI provides guidance when capabilities are missing
	GenerateCapabilityGuidance(ctx context.Context, gaps []*AgentCapabilityGap) (string, error)
}

// Example AI-driven agent discovery:
// User: "Create a sentiment analysis test plan and store it in GitHub"
// AI discovers:
//   - Need: NLTK/NLP capabilities → Found: NLTK Research Agent ✅
//   - Need: Test planning → Found: Python Test Agent ✅
//   - Need: GitHub integration → Found: NO AGENTS ❌
// Result: "I can help with sentiment analysis and test planning, but you need to register a GitHub agent first"
