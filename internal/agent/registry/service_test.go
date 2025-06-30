package registry_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"neuromesh/internal/agent/domain"
	"neuromesh/internal/agent/registry"
	"neuromesh/internal/logging"
	"neuromesh/testHelpers"
)

func TestAgentRegistry_RegisterAgent_Success(t *testing.T) {
	// Arrange
	ctx := context.Background()
	logger := logging.NewStructuredLogger(logging.LevelError) // Reduce noise in tests

	// Create mock graph for testing
	testGraph := testHelpers.NewCleanMockGraph()

	registryService := registry.NewService(testGraph, logger)

	// Create test agent
	agent := &domain.Agent{
		ID:          "test-agent-1",
		Name:        "Test Agent",
		Description: "A test agent for unit testing",
		Status:      domain.AgentStatusOnline,
		Capabilities: []domain.AgentCapability{
			{
				Name:        "text-processing",
				Description: "Can process text",
				Parameters:  map[string]string{"model": "gpt-4"},
			},
		},
		Metadata:  map[string]string{"env": "test"},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		LastSeen:  time.Now(),
	}

	// Act
	err := registryService.RegisterAgent(ctx, agent)

	// Assert
	assert.NoError(t, err)

	// Verify agent was registered
	retrievedAgent, err := registryService.GetAgent(ctx, agent.ID)
	require.NoError(t, err)
	assert.Equal(t, agent.ID, retrievedAgent.ID)
	assert.Equal(t, agent.Name, retrievedAgent.Name)
	assert.Equal(t, agent.Status, retrievedAgent.Status)
}

func TestAgentRegistry_RegisterAgent_ValidationErrors(t *testing.T) {
	// Arrange
	ctx := context.Background()
	logger := logging.NewStructuredLogger(logging.LevelError)

	testGraph := testHelpers.NewCleanMockGraph()

	registryService := registry.NewService(testGraph, logger)

	tests := []struct {
		name        string
		agent       *domain.Agent
		expectedErr string
	}{
		{
			name:        "nil agent",
			agent:       nil,
			expectedErr: "agent cannot be nil",
		},
		{
			name: "empty ID",
			agent: &domain.Agent{
				Name: "Test Agent",
			},
			expectedErr: "agent ID cannot be empty",
		},
		{
			name: "empty name",
			agent: &domain.Agent{
				ID: "test-id",
			},
			expectedErr: "agent name cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			err := registryService.RegisterAgent(ctx, tt.agent)

			// Assert
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedErr)
		})
	}
}

func TestAgentRegistry_GetAgentsByCapability(t *testing.T) {
	// Arrange
	ctx := context.Background()
	logger := logging.NewStructuredLogger(logging.LevelError)

	testGraph := testHelpers.NewCleanMockGraph()

	registryService := registry.NewService(testGraph, logger)

	// Register multiple agents with different capabilities
	agents := []*domain.Agent{
		{
			ID:     "agent-1",
			Name:   "Text Processor",
			Status: domain.AgentStatusOnline,
			Capabilities: []domain.AgentCapability{
				{Name: "text-processing", Description: "Process text"},
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:     "agent-2",
			Name:   "Image Processor",
			Status: domain.AgentStatusOnline,
			Capabilities: []domain.AgentCapability{
				{Name: "image-processing", Description: "Process images"},
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:     "agent-3",
			Name:   "Multi Processor",
			Status: domain.AgentStatusOnline,
			Capabilities: []domain.AgentCapability{
				{Name: "text-processing", Description: "Process text"},
				{Name: "image-processing", Description: "Process images"},
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	for _, agent := range agents {
		err := registryService.RegisterAgent(ctx, agent)
		require.NoError(t, err)
	}

	// Act
	textProcessors, err := registryService.GetAgentsByCapability(ctx, "text-processing")

	// Assert
	require.NoError(t, err)
	assert.Len(t, textProcessors, 2) // agent-1 and agent-3

	agentIDs := make([]string, len(textProcessors))
	for i, agent := range textProcessors {
		agentIDs[i] = agent.ID
	}
	assert.Contains(t, agentIDs, "agent-1")
	assert.Contains(t, agentIDs, "agent-3")
}

func TestAgentRegistry_UpdateAgentStatus(t *testing.T) {
	// Arrange
	ctx := context.Background()
	logger := logging.NewStructuredLogger(logging.LevelError)

	testGraph := testHelpers.NewCleanMockGraph()

	registryService := registry.NewService(testGraph, logger)

	// Register an agent
	agent := &domain.Agent{
		ID:        "test-agent",
		Name:      "Test Agent",
		Status:    domain.AgentStatusOnline,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := registryService.RegisterAgent(ctx, agent)
	require.NoError(t, err)

	// Act - Update status
	err = registryService.UpdateAgentStatus(ctx, agent.ID, domain.AgentStatusBusy)

	// Assert
	require.NoError(t, err)

	// Verify status was updated
	updatedAgent, err := registryService.GetAgent(ctx, agent.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.AgentStatusBusy, updatedAgent.Status)
}

func TestAgentRegistry_IsAgentHealthy(t *testing.T) {
	// Arrange
	ctx := context.Background()
	logger := logging.NewStructuredLogger(logging.LevelError)

	testGraph := testHelpers.NewCleanMockGraph()

	registryService := registry.NewService(testGraph, logger)

	// Register an agent
	agent := &domain.Agent{
		ID:        "healthy-agent",
		Name:      "Healthy Agent",
		Status:    domain.AgentStatusOnline,
		LastSeen:  time.Now(), // Recent last seen
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := registryService.RegisterAgent(ctx, agent)
	require.NoError(t, err)

	// Act
	isHealthy, err := registryService.IsAgentHealthy(ctx, agent.ID)

	// Assert
	require.NoError(t, err)
	assert.True(t, isHealthy)
}

// TDD RED: Test for 30-second agent health monitoring requirement
func TestAgentRegistry_IsAgentHealthy_ThirtySecondTimeout(t *testing.T) {
	// Arrange
	ctx := context.Background()
	logger := logging.NewStructuredLogger(logging.LevelError)
	testGraph := testHelpers.NewCleanMockGraph()
	registryService := registry.NewService(testGraph, logger)

	// Create agent with last seen 31 seconds ago (should be unhealthy)
	agentID := "test-agent-timeout"
	agent := &domain.Agent{
		ID:          agentID,
		Name:        "Timeout Test Agent",
		Description: "Agent for testing 30-second timeout",
		Status:      domain.AgentStatusOnline,
		Capabilities: []domain.AgentCapability{
			{Name: "test", Description: "Test capability"},
		},
		CreatedAt: time.Now().Add(-2 * time.Minute),
		UpdatedAt: time.Now().Add(-31 * time.Second), // 31 seconds ago
		LastSeen:  time.Now().Add(-31 * time.Second), // 31 seconds ago - should be unhealthy
	}

	// Register the agent
	err := registryService.RegisterAgent(ctx, agent)
	require.NoError(t, err)

	// Act
	isHealthy, err := registryService.IsAgentHealthy(ctx, agentID)

	// Assert - TDD RED: This should fail with current 5-minute implementation
	assert.NoError(t, err)
	assert.False(t, isHealthy, "Agent should be unhealthy after 31 seconds without heartbeat")

	// Test edge case: exactly 30 seconds should still be healthy
	agent.LastSeen = time.Now().Add(-30 * time.Second)
	agent.UpdatedAt = time.Now().Add(-30 * time.Second)
	err = registryService.RegisterAgent(ctx, agent) // Re-register with updated timestamp
	require.NoError(t, err)

	isHealthy, err = registryService.IsAgentHealthy(ctx, agentID)
	assert.NoError(t, err)
	assert.True(t, isHealthy, "Agent should still be healthy at exactly 30 seconds")

	// Test recently updated agent should be healthy
	agent.LastSeen = time.Now().Add(-10 * time.Second)
	agent.UpdatedAt = time.Now().Add(-10 * time.Second)
	err = registryService.RegisterAgent(ctx, agent) // Re-register with recent timestamp
	require.NoError(t, err)

	isHealthy, err = registryService.IsAgentHealthy(ctx, agentID)
	assert.NoError(t, err)
	assert.True(t, isHealthy, "Agent should be healthy with recent heartbeat")
}

func TestAgentRegistry_MonitorAgentHealth_AutoDisconnect(t *testing.T) {
	// Arrange
	ctx := context.Background()
	logger := logging.NewStructuredLogger(logging.LevelError)
	testGraph := testHelpers.NewCleanMockGraph()
	registryService := registry.NewService(testGraph, logger)

	// Create agent that will become unhealthy
	agentID := "test-agent-monitor"
	agent := &domain.Agent{
		ID:          agentID,
		Name:        "Monitor Test Agent",
		Description: "Agent for testing health monitoring",
		Status:      domain.AgentStatusOnline,
		Capabilities: []domain.AgentCapability{
			{Name: "test", Description: "Test capability"},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		LastSeen:  time.Now().Add(-35 * time.Second), // Unhealthy
	}

	err := registryService.RegisterAgent(ctx, agent)
	require.NoError(t, err)

	// Act - TDD RED: This method doesn't exist yet, should fail
	err = registryService.MonitorAgentHealth(ctx)

	// Assert
	assert.NoError(t, err, "MonitorAgentHealth should execute without error")

	// Verify agent status was updated to Disconnected
	updatedAgent, err := registryService.GetAgent(ctx, agentID)
	require.NoError(t, err)
	assert.Equal(t, domain.AgentStatusDisconnected, updatedAgent.Status,
		"Agent should be marked as Disconnected after health monitoring")
}

// Interface compliance test
func TestAgentRegistry_ImplementsInterface(t *testing.T) {
	// Arrange
	logger := logging.NewStructuredLogger(logging.LevelError)
	testGraph := testHelpers.NewCleanMockGraph()

	// Act & Assert - This will fail to compile if Service doesn't implement AgentRegistry
	var _ domain.AgentRegistry = registry.NewService(testGraph, logger)
}
