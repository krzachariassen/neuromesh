package infrastructure

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"neuromesh/internal/planning/domain"
	"neuromesh/testHelpers"
)

// Unit tests using mocks for TDD RED/GREEN/REFACTOR cycle
// These tests focus on repository logic without external dependencies

func TestGraphAnalysisRepository_Store_Unit(t *testing.T) {
	t.Run("RED: should call AddNode and AddEdge with correct parameters", func(t *testing.T) {
		// Setup mock graph
		mockGraph := testHelpers.NewTestifyMockGraph()
		repo := NewGraphAnalysisRepository(mockGraph)

		// Create test analysis
		analysis := domain.NewAnalysis(
			"test-request-123",
			"deploy_app",
			"deployment",
			85,
			[]string{"deploy-agent"},
			"User wants to deploy application",
		)

		// Expected calls - this defines the contract for the repository
		// We use mock.Anything for properties since exact matching is complex

		// Set up mock expectations - use mock.Anything for complex properties
		mockGraph.(*testHelpers.TestifyMockGraph).On("AddNode",
			context.Background(),
			"Analysis",
			analysis.ID,
			mock.Anything).Return(nil)

		mockGraph.(*testHelpers.TestifyMockGraph).On("AddEdge",
			context.Background(),
			"Message",
			analysis.RequestID,
			"Analysis",
			analysis.ID,
			"TRIGGERS_ANALYSIS",
			mock.Anything).Return(nil)

		// Execute - THIS WILL FAIL until we implement Store correctly
		err := repo.Store(context.Background(), analysis)

		// Assertions
		require.NoError(t, err)
		mockGraph.(*testHelpers.TestifyMockGraph).AssertExpectations(t)
	})

	t.Run("RED: should handle AddNode error gracefully", func(t *testing.T) {
		mockGraph := testHelpers.NewTestifyMockGraph()
		repo := NewGraphAnalysisRepository(mockGraph)

		analysis := domain.NewAnalysis(
			"test-request-456",
			"monitor_system",
			"monitoring",
			75,
			[]string{"monitor-agent"},
			"System monitoring request",
		)

		// Mock AddNode to return error
		mockGraph.(*testHelpers.TestifyMockGraph).On("AddNode",
			context.Background(),
			"Analysis",
			analysis.ID,
			mock.Anything).Return(assert.AnError)

		// Execute - should handle error
		err := repo.Store(context.Background(), analysis)

		// Should return the error
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create Analysis node")
		mockGraph.(*testHelpers.TestifyMockGraph).AssertExpectations(t)
	})
}

func TestGraphAnalysisRepository_GetByID_Unit(t *testing.T) {
	t.Run("RED: should query node by ID and convert to Analysis", func(t *testing.T) {
		mockGraph := testHelpers.NewTestifyMockGraph()
		repo := NewGraphAnalysisRepository(mockGraph)

		analysisID := "test-analysis-789"
		expectedNodeData := map[string]interface{}{
			"id":              analysisID,
			"request_id":      "test-message-123",
			"intent":          "deploy_service",
			"category":        "deployment",
			"confidence":      int64(90), // Neo4j returns int64
			"required_agents": mustMarshalJSON([]string{"deploy-agent", "test-agent"}),
			"reasoning":       "Clear deployment intent detected",
			"timestamp":       "2025-01-01T10:00:00Z",
		}

		// Mock QueryNodes to return the test data
		mockGraph.(*testHelpers.TestifyMockGraph).On("QueryNodes",
			context.Background(),
			"Analysis",
			map[string]interface{}{"id": analysisID}).Return([]map[string]interface{}{expectedNodeData}, nil)

		// Execute - THIS WILL FAIL until we implement GetByID correctly
		result, err := repo.GetByID(context.Background(), analysisID)

		// Assertions
		require.NoError(t, err)
		assert.Equal(t, analysisID, result.ID)
		assert.Equal(t, "test-message-123", result.RequestID)
		assert.Equal(t, "deploy_service", result.Intent)
		assert.Equal(t, "deployment", result.Category)
		assert.Equal(t, 90, result.Confidence)
		assert.Equal(t, []string{"deploy-agent", "test-agent"}, result.RequiredAgents)
		assert.Equal(t, "Clear deployment intent detected", result.Reasoning)
		mockGraph.(*testHelpers.TestifyMockGraph).AssertExpectations(t)
	})

	t.Run("RED: should return error when analysis not found", func(t *testing.T) {
		mockGraph := testHelpers.NewTestifyMockGraph()
		repo := NewGraphAnalysisRepository(mockGraph)

		analysisID := "nonexistent-analysis"

		// Mock QueryNodes to return empty result
		mockGraph.(*testHelpers.TestifyMockGraph).On("QueryNodes",
			context.Background(),
			"Analysis",
			map[string]interface{}{"id": analysisID}).Return([]map[string]interface{}{}, nil)

		// Execute
		result, err := repo.GetByID(context.Background(), analysisID)

		// Should return error for not found
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "analysis not found")
		mockGraph.(*testHelpers.TestifyMockGraph).AssertExpectations(t)
	})
}

func TestGraphAnalysisRepository_GetByRequestID_Unit(t *testing.T) {
	t.Run("RED: should query by request_id and return Analysis", func(t *testing.T) {
		mockGraph := testHelpers.NewTestifyMockGraph()
		repo := NewGraphAnalysisRepository(mockGraph)

		requestID := "test-message-456"
		nodeData := map[string]interface{}{
			"id":              "analysis-123",
			"request_id":      requestID,
			"intent":          "security_scan",
			"category":        "security",
			"confidence":      int64(80),
			"required_agents": mustMarshalJSON([]string{"security-agent"}),
			"reasoning":       "Security analysis needed",
			"timestamp":       "2025-01-01T11:00:00Z",
		}

		mockGraph.(*testHelpers.TestifyMockGraph).On("QueryNodes",
			context.Background(),
			"Analysis",
			map[string]interface{}{"request_id": requestID}).Return([]map[string]interface{}{nodeData}, nil)

		// Execute
		result, err := repo.GetByRequestID(context.Background(), requestID)

		// Assertions
		require.NoError(t, err)
		assert.Equal(t, "analysis-123", result.ID)
		assert.Equal(t, requestID, result.RequestID)
		assert.Equal(t, "security_scan", result.Intent)
		mockGraph.(*testHelpers.TestifyMockGraph).AssertExpectations(t)
	})
}

func TestGraphAnalysisRepository_GetByUserID_Unit(t *testing.T) {
	t.Run("RED: should retrieve analyses by user with limit", func(t *testing.T) {
		mockGraph := testHelpers.NewTestifyMockGraph()
		repo := NewGraphAnalysisRepository(mockGraph)

		userID := "test-user-123"
		limit := 10

		// Mock QueryNodes to return empty for now (since GetByUserID has TODO for user filtering)
		mockGraph.(*testHelpers.TestifyMockGraph).On("QueryNodes",
			context.Background(),
			"Analysis",
			map[string]interface{}{}).Return([]map[string]interface{}{}, nil)

		// Execute
		results, err := repo.GetByUserID(context.Background(), userID, limit)

		// Should work without error (even if not filtering by user yet)
		require.NoError(t, err)
		assert.Empty(t, results) // Should return empty slice for no data
		mockGraph.(*testHelpers.TestifyMockGraph).AssertExpectations(t)
	})
}

func TestGraphAnalysisRepository_GetByConfidenceRange_Unit(t *testing.T) {
	t.Run("RED: should filter analyses by confidence range", func(t *testing.T) {
		mockGraph := testHelpers.NewTestifyMockGraph()
		repo := NewGraphAnalysisRepository(mockGraph)

		// Mock data with analyses of different confidence levels
		mockData := []map[string]interface{}{
			{
				"id":              "analysis-low",
				"request_id":      "msg-1",
				"intent":          "unclear",
				"category":        "general",
				"confidence":      int64(30),
				"required_agents": mustMarshalJSON([]string{}),
				"reasoning":       "Low confidence",
				"timestamp":       "2025-01-01T10:00:00Z",
			},
			{
				"id":              "analysis-high",
				"request_id":      "msg-2",
				"intent":          "deploy",
				"category":        "deployment",
				"confidence":      int64(95),
				"required_agents": mustMarshalJSON([]string{"deploy-agent"}),
				"reasoning":       "High confidence",
				"timestamp":       "2025-01-01T11:00:00Z",
			},
		}

		mockGraph.(*testHelpers.TestifyMockGraph).On("QueryNodes",
			context.Background(),
			"Analysis",
			map[string]interface{}{}).Return(mockData, nil)

		// Execute - get analyses with confidence 80-100
		results, err := repo.GetByConfidenceRange(context.Background(), 80, 100, 10)

		// Should filter to only high confidence analysis
		require.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, "analysis-high", results[0].ID)
		assert.Equal(t, 95, results[0].Confidence)
		mockGraph.(*testHelpers.TestifyMockGraph).AssertExpectations(t)
	})
}

func TestGraphAnalysisRepository_GetByCategory_Unit(t *testing.T) {
	t.Run("RED: should query analyses by specific category", func(t *testing.T) {
		mockGraph := testHelpers.NewTestifyMockGraph()
		repo := NewGraphAnalysisRepository(mockGraph)

		category := "deployment"
		mockData := []map[string]interface{}{
			{
				"id":              "analysis-deploy-1",
				"request_id":      "msg-1",
				"intent":          "deploy_app",
				"category":        category,
				"confidence":      int64(85),
				"required_agents": mustMarshalJSON([]string{"deploy-agent"}),
				"reasoning":       "Deploy request",
				"timestamp":       "2025-01-01T10:00:00Z",
			},
		}

		mockGraph.(*testHelpers.TestifyMockGraph).On("QueryNodes",
			context.Background(),
			"Analysis",
			map[string]interface{}{"category": category}).Return(mockData, nil)

		// Execute
		results, err := repo.GetByCategory(context.Background(), category, 10)

		// Should return deployment analyses
		require.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, "analysis-deploy-1", results[0].ID)
		assert.Equal(t, category, results[0].Category)
		mockGraph.(*testHelpers.TestifyMockGraph).AssertExpectations(t)
	})
}

// Helper functions for test expectations

func mustMarshalJSON(v interface{}) string {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(data)
}
