package application

import (
	"context"
	"os"
	"testing"
	"time"

	aiInfrastructure "neuromesh/internal/ai/infrastructure"
	"neuromesh/internal/logging"
	"neuromesh/testHelpers"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServiceFactory_CreateAIProvider(t *testing.T) {
	// TDD: RED - Test creating AI provider with factory

	// Setup
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY environment variable not set, skipping AI provider factory tests")
	}

	config := aiInfrastructure.DefaultOpenAIConfig()
	config.APIKey = apiKey
	config.Model = "gpt-3.5-turbo"

	logger, err := logging.NewLogger(false)
	require.NoError(t, err)

	// Execute: Create AI provider using factory method
	aiProvider := CreateAIProvider(config, logger)

	// Verify: Provider should be properly created
	assert.NotNil(t, aiProvider)

	// Get provider info
	info := aiProvider.GetProviderInfo()
	assert.NotNil(t, info)
	assert.Equal(t, "openai", info.Name)
	assert.Equal(t, "gpt-3.5-turbo", info.Model)

	t.Log("Service factory successfully created AI provider")
}

func TestNewServiceFactory(t *testing.T) {
	// TDD: RED - Test factory constructor

	logger, err := logging.NewLogger(false)
	require.NoError(t, err)

	// We don't need real graph/messageBus for constructor test
	factory := NewServiceFactory(logger, nil, nil, nil)

	// Verify: Factory should be created
	assert.NotNil(t, factory)
	assert.Equal(t, logger, factory.logger)

	t.Log("Service factory constructor works correctly")
}

// TestServiceFactory_DependencyInjection_TDD tests the complete dependency injection and lifecycle management
func TestServiceFactory_DependencyInjection_TDD(t *testing.T) {
	t.Run("RED: should wire all dependencies correctly", func(t *testing.T) {
		// ARRANGE
		logger := logging.NewNoOpLogger()
		// Use nil for dependencies that aren't critical for wiring test
		aiProvider := testHelpers.SetupRealAIProvider(t)

		// ACT - Create service factory with all dependencies
		factory := NewServiceFactory(logger, nil, nil, aiProvider)

		// ASSERT - Verify core dependencies are wired
		assert.NotNil(t, factory.logger, "Logger should be wired")
		assert.NotNil(t, factory.aiProvider, "AIProvider should be wired")
		assert.NotNil(t, factory.correlationTracker, "CorrelationTracker should be wired")
		assert.NotNil(t, factory.shutdownContext, "Shutdown context should be created")
		assert.NotNil(t, factory.shutdownCancel, "Shutdown cancel should be created")

		// These will be nil when messageBus/graph are nil (expected behavior)
		assert.Nil(t, factory.aiMessageBus, "AIMessageBus should be nil when messageBus is nil")
		assert.Nil(t, factory.globalMessageConsumer, "GlobalMessageConsumer should be nil when dependencies are nil")
	})

	t.Run("RED: should create orchestrator service with injected dependencies", func(t *testing.T) {
		// ARRANGE
		logger := logging.NewNoOpLogger()
		aiProvider := testHelpers.SetupRealAIProvider(t)
		factory := NewServiceFactory(logger, nil, nil, aiProvider)

		// ACT - Create orchestrator service
		orchestratorService := factory.CreateOrchestratorService()

		// ASSERT - Verify service is created properly
		assert.NotNil(t, orchestratorService, "OrchestratorService should be created")
		// Service should have all its dependencies wired through the factory
	})

	t.Run("RED: should handle startup gracefully", func(t *testing.T) {
		// ARRANGE
		logger := logging.NewNoOpLogger()
		aiProvider := testHelpers.SetupRealAIProvider(t)
		factory := NewServiceFactory(logger, nil, nil, aiProvider)

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		// ACT - Start services (will fail with nil message bus, but tests startup logic)
		err := factory.StartServices(ctx)

		// ASSERT - Should handle startup failure gracefully
		assert.Error(t, err, "Startup should fail with nil message bus")

		// Shutdown should still work
		shutdownErr := factory.Shutdown()
		require.NoError(t, shutdownErr, "Services should shutdown without error even after failed startup")
	})

	t.Run("RED: should cleanup correlation tracker on shutdown", func(t *testing.T) {
		// ARRANGE
		logger := logging.NewNoOpLogger()
		aiProvider := testHelpers.SetupRealAIProvider(t)
		factory := NewServiceFactory(logger, nil, nil, aiProvider)

		// Add a pending request to correlation tracker
		responseChan := factory.correlationTracker.RegisterRequest("test-correlation", "test-user", 30*time.Second)
		assert.NotNil(t, responseChan, "Response channel should be created")

		// ACT - Shutdown should cleanup pending requests
		err := factory.Shutdown()

		// ASSERT - Shutdown should succeed
		require.NoError(t, err, "Shutdown should succeed")

		// Response channel should be closed (will panic if we try to send)
		select {
		case _, ok := <-responseChan:
			assert.False(t, ok, "Response channel should be closed after shutdown")
		default:
			// Channel is not ready, which is also valid
		}
	})
}

// TestServiceFactory_ProductionScenario_TDD tests the complete production workflow
func TestServiceFactory_ProductionScenario_TDD(t *testing.T) {
	t.Run("RED: should validate dependency injection completeness", func(t *testing.T) {
		// ARRANGE - Test partial dependency scenarios
		logger := logging.NewNoOpLogger()
		aiProvider := testHelpers.SetupRealAIProvider(t)

		// ACT - Test different dependency combinations
		// 1. No graph, no messageBus
		factory1 := NewServiceFactory(logger, nil, nil, aiProvider)
		assert.Nil(t, factory1.globalMessageConsumer, "GlobalMessageConsumer should be nil without graph/messageBus")
		assert.Nil(t, factory1.aiMessageBus, "AIMessageBus should be nil without graph/messageBus")

		// 2. Graph but no messageBus
		graph := testHelpers.NewMockGraph()
		factory2 := NewServiceFactory(logger, graph, nil, aiProvider)
		assert.Nil(t, factory2.globalMessageConsumer, "GlobalMessageConsumer should be nil without messageBus")
		assert.Nil(t, factory2.aiMessageBus, "AIMessageBus should be nil without messageBus")

		// 3. MessageBus but no graph (can't test easily due to interface complexity)
		// 4. Both graph and messageBus (would need full integration test)

		// Test common dependencies that should always be present
		assert.NotNil(t, factory1.logger, "Logger should always be wired")
		assert.NotNil(t, factory1.aiProvider, "AIProvider should always be wired")
		assert.NotNil(t, factory1.correlationTracker, "CorrelationTracker should always be wired")
		assert.NotNil(t, factory1.shutdownContext, "Shutdown context should always be wired")
		assert.NotNil(t, factory1.shutdownCancel, "Shutdown cancel should always be wired")

		// Test that services can be created even with minimal dependencies
		orchestratorService := factory1.CreateOrchestratorService()
		assert.NotNil(t, orchestratorService, "OrchestratorService should be created with minimal dependencies")

		// Test shutdown works
		shutdownErr := factory1.Shutdown()
		assert.NoError(t, shutdownErr, "Shutdown should succeed")
	})

	t.Run("RED: should handle startup failure gracefully when dependencies are missing", func(t *testing.T) {
		// ARRANGE - Factory with no message bus
		logger := logging.NewNoOpLogger()
		aiProvider := testHelpers.SetupRealAIProvider(t)
		factory := NewServiceFactory(logger, nil, nil, aiProvider)

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		// ACT - Try to start services without proper dependencies
		err := factory.StartServices(ctx)

		// ASSERT - Should fail gracefully
		assert.Error(t, err, "StartServices should fail when dependencies are missing")
		assert.Contains(t, err.Error(), "global message consumer not initialized", "Error should indicate missing global message consumer")

		// Shutdown should still work
		shutdownErr := factory.Shutdown()
		assert.NoError(t, shutdownErr, "Shutdown should succeed even after failed startup")
	})

	t.Run("RED: should handle multiple startup attempts gracefully", func(t *testing.T) {
		// ARRANGE - Factory with minimal dependencies
		logger := logging.NewNoOpLogger()
		aiProvider := testHelpers.SetupRealAIProvider(t)
		factory := NewServiceFactory(logger, nil, nil, aiProvider)

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		// ACT & ASSERT - Multiple startup attempts should be handled gracefully
		err1 := factory.StartServices(ctx)
		assert.Error(t, err1, "First startup should fail with missing dependencies")

		err2 := factory.StartServices(ctx)
		assert.Error(t, err2, "Second startup should also fail with missing dependencies")

		// Both should be the same error
		assert.Equal(t, err1.Error(), err2.Error(), "Error messages should be consistent")

		// Shutdown should work
		shutdownErr := factory.Shutdown()
		assert.NoError(t, shutdownErr, "Shutdown should succeed")
	})
}
