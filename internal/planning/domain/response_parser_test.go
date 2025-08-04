package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// NOTE: Capability gap extraction functionality removed - AI handles capability analysis in reasoning.
// The AgentGap field is now set to empty since AI provides all capability gap analysis
// in the reasoning field which is more valuable and user-friendly.

func TestResponseParser_ExtractSection(t *testing.T) {
	parser := NewResponseParser()

	t.Run("should extract section from AI response", func(t *testing.T) {
		aiResponse := `PLANNING_RESULT:
Intent: test_intent
Category: test_category
Confidence: 95
Reasoning: This is the reasoning section`

		intent := parser.ExtractSection(aiResponse, "Intent:")
		assert.Equal(t, "test_intent", intent)

		reasoning := parser.ExtractSection(aiResponse, "Reasoning:")
		assert.Equal(t, "This is the reasoning section", reasoning)
	})

	t.Run("should return empty string if section not found", func(t *testing.T) {
		aiResponse := `PLANNING_RESULT:
Intent: test_intent`

		missing := parser.ExtractSection(aiResponse, "NotFound:")
		assert.Equal(t, "", missing)
	})
}

func TestResponseParser_ParseConfidence(t *testing.T) {
	parser := NewResponseParser()

	t.Run("should parse confidence from string", func(t *testing.T) {
		confidence := parser.ParseConfidence("85")
		assert.Equal(t, 85, confidence)

		confidence = parser.ParseConfidence("Confidence: 95")
		assert.Equal(t, 95, confidence)
	})

	t.Run("should return 0 for invalid confidence", func(t *testing.T) {
		confidence := parser.ParseConfidence("invalid")
		assert.Equal(t, 0, confidence)

		confidence = parser.ParseConfidence("")
		assert.Equal(t, 0, confidence)
	})
}

func TestResponseParser_ExtractIntent(t *testing.T) {
	parser := NewResponseParser()

	t.Run("should extract and normalize intent", func(t *testing.T) {
		analysis := "Intent: Process Text Data\nCategory: text_processing"
		intent := parser.ExtractIntent(analysis)
		assert.Equal(t, "process_text_data", intent)
	})

	t.Run("should return default intent if not found", func(t *testing.T) {
		analysis := "Category: text_processing"
		intent := parser.ExtractIntent(analysis)
		assert.Equal(t, "general_assistance", intent)
	})
}

func TestResponseParser_ExtractCategory(t *testing.T) {
	parser := NewResponseParser()

	t.Run("should extract and normalize category", func(t *testing.T) {
		analysis := "Intent: test\nCategory: Text Processing"
		category := parser.ExtractCategory(analysis)
		assert.Equal(t, "text_processing", category)
	})

	t.Run("should return default category if not found", func(t *testing.T) {
		analysis := "Intent: test"
		category := parser.ExtractCategory(analysis)
		assert.Equal(t, "general", category)
	})
}

func TestResponseParser_ExtractRequiredAgents(t *testing.T) {
	parser := NewResponseParser()

	t.Run("should parse comma-separated agents", func(t *testing.T) {
		analysis := "Required_Agents: agent1, agent2, agent3"
		agents := parser.ExtractRequiredAgents(analysis)
		assert.Equal(t, []string{"agent1", "agent2", "agent3"}, agents)
	})

	t.Run("should handle single agent", func(t *testing.T) {
		analysis := "Required_Agents: single-agent"
		agents := parser.ExtractRequiredAgents(analysis)
		assert.Equal(t, []string{"single-agent"}, agents)
	})

	t.Run("should return empty slice if no agents", func(t *testing.T) {
		analysis := "Intent: test"
		agents := parser.ExtractRequiredAgents(analysis)
		assert.Empty(t, agents)
	})
}
