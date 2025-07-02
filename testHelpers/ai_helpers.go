package testHelpers

import (
	"os"
	"testing"

	aiInfrastructure "neuromesh/internal/ai/infrastructure"
	"neuromesh/internal/logging"
)

// SetupRealAIProvider creates a real OpenAI provider for testing
// This function should be used across all tests that need a real AI provider
// following TDD principles (no mocking of AI behavior)
func SetupRealAIProvider(t *testing.T) *aiInfrastructure.OpenAIProvider {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY environment variable not set, skipping AI provider tests")
	}

	config := aiInfrastructure.DefaultOpenAIConfig()
	config.APIKey = apiKey
	config.Model = "gpt-4.1-mini" // Use faster model for tests
	config.MaxTokens = 1000       // Limit tokens for faster tests

	logger := logging.NewNoOpLogger() // Use no-op logger for tests
	provider := aiInfrastructure.NewOpenAIProvider(config, logger)

	return provider
}
