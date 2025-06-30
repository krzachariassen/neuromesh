package testHelpers

import (
	"context"

	"github.com/stretchr/testify/mock"
	"neuromesh/internal/graph"
)

// MockGraph provides a simple in-memory graph for testing
type MockGraph struct {
	nodes map[string]map[string]interface{}
}

// NewMockGraph creates a new mock graph instance with realistic test data
func NewMockGraph() graph.Graph {
	mockGraph := &MockGraph{
		nodes: make(map[string]map[string]interface{}),
	}

	// Add realistic test data for AI tests
	mockGraph.addTestData()

	return mockGraph
}

// NewCleanMockGraph creates a new mock graph instance without any test data
// Use this for tests that need to control the exact graph state
func NewCleanMockGraph() graph.Graph {
	return &MockGraph{
		nodes: make(map[string]map[string]interface{}),
	}
}

// addTestData populates the mock graph with realistic test data
func (m *MockGraph) addTestData() {
	// Add test agents
	m.nodes["agent:deployment_agent"] = map[string]interface{}{
		"id":           "deployment_agent",
		"type":         "agent",
		"name":         "Deployment Specialist",
		"capabilities": []string{"deploy", "rollback", "canary"},
		"status":       "active",
		"description":  "Handles application deployments and rollbacks",
	}

	m.nodes["agent:security_guardian"] = map[string]interface{}{
		"id":           "security_guardian",
		"type":         "agent",
		"name":         "Security Guardian",
		"capabilities": []string{"security", "compliance", "audit"},
		"status":       "active",
		"description":  "Enforces security policies and compliance",
	}

	m.nodes["agent:performance_optimizer"] = map[string]interface{}{
		"id":           "performance_optimizer",
		"type":         "agent",
		"name":         "Performance Optimizer",
		"capabilities": []string{"monitoring", "optimization", "scaling"},
		"status":       "active",
		"description":  "Monitors and optimizes system performance",
	}

	// Add test workflows
	m.nodes["workflow:deploy_app"] = map[string]interface{}{
		"id":          "deploy_app",
		"type":        "workflow",
		"name":        "Deploy Application",
		"description": "Standard application deployment workflow",
		"steps":       []string{"validate", "build", "test", "deploy"},
		"status":      "active",
	}

	// Add test conversations
	m.nodes["conversation:user123_001"] = map[string]interface{}{
		"id":        "user123_001",
		"type":      "conversation",
		"user_id":   "user123",
		"timestamp": "2024-01-15T10:30:00Z",
		"request":   "Deploy my application to production",
		"response":  "Deployment completed successfully",
		"status":    "completed",
	}
}

// TestifyMockGraph provides a testify-based mock for complex test scenarios
type TestifyMockGraph struct {
	mock.Mock
}

// NewTestifyMockGraph creates a new testify-based mock graph
func NewTestifyMockGraph() graph.Graph {
	return &TestifyMockGraph{}
}

func (m *TestifyMockGraph) AddNode(ctx context.Context, nodeType, nodeID string, properties map[string]interface{}) error {
	args := m.Called(ctx, nodeType, nodeID, properties)
	return args.Error(0)
}

func (m *TestifyMockGraph) GetNode(ctx context.Context, nodeType, nodeID string) (map[string]interface{}, error) {
	args := m.Called(ctx, nodeType, nodeID)
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *TestifyMockGraph) UpdateNode(ctx context.Context, nodeType, nodeID string, properties map[string]interface{}) error {
	args := m.Called(ctx, nodeType, nodeID, properties)
	return args.Error(0)
}

func (m *TestifyMockGraph) DeleteNode(ctx context.Context, nodeType, nodeID string) error {
	args := m.Called(ctx, nodeType, nodeID)
	return args.Error(0)
}

func (m *TestifyMockGraph) QueryNodes(ctx context.Context, nodeType string, filters map[string]interface{}) ([]map[string]interface{}, error) {
	args := m.Called(ctx, nodeType, filters)
	return args.Get(0).([]map[string]interface{}), args.Error(1)
}

func (m *TestifyMockGraph) GetStats() map[string]interface{} {
	args := m.Called()
	return args.Get(0).(map[string]interface{})
}

// TestifyMockGraph missing methods
func (m *TestifyMockGraph) AddEdge(ctx context.Context, sourceType, sourceID, targetType, targetID, edgeType string, properties map[string]interface{}) error {
	args := m.Called(ctx, sourceType, sourceID, targetType, targetID, edgeType, properties)
	return args.Error(0)
}

func (m *TestifyMockGraph) GetEdges(ctx context.Context, nodeType, nodeID string) ([]map[string]interface{}, error) {
	args := m.Called(ctx, nodeType, nodeID)
	return args.Get(0).([]map[string]interface{}), args.Error(1)
}

func (m *TestifyMockGraph) UpdateEdge(ctx context.Context, sourceType, sourceID, targetType, targetID, edgeType string, properties map[string]interface{}) error {
	args := m.Called(ctx, sourceType, sourceID, targetType, targetID, edgeType, properties)
	return args.Error(0)
}

func (m *TestifyMockGraph) DeleteEdge(ctx context.Context, sourceType, sourceID, targetType, targetID, edgeType string) error {
	args := m.Called(ctx, sourceType, sourceID, targetType, targetID, edgeType)
	return args.Error(0)
}

func (m *TestifyMockGraph) Close(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// AddNode adds a node to the mock graph
func (m *MockGraph) AddNode(ctx context.Context, nodeType, nodeID string, properties map[string]interface{}) error {
	key := nodeType + ":" + nodeID
	if properties == nil {
		properties = make(map[string]interface{})
	}
	properties["id"] = nodeID
	properties["type"] = nodeType
	m.nodes[key] = properties
	return nil
}

// GetNode retrieves a node from the mock graph
func (m *MockGraph) GetNode(ctx context.Context, nodeType, nodeID string) (map[string]interface{}, error) {
	key := nodeType + ":" + nodeID
	if props, exists := m.nodes[key]; exists {
		return props, nil
	}
	return nil, nil // Return nil, nil for not found (compatible with registry tests)
}

// UpdateNode updates a node in the mock graph
func (m *MockGraph) UpdateNode(ctx context.Context, nodeType, nodeID string, properties map[string]interface{}) error {
	key := nodeType + ":" + nodeID
	if existing, exists := m.nodes[key]; exists {
		for k, v := range properties {
			existing[k] = v
		}
	}
	return nil // Always return success (compatible with registry tests)
}

// DeleteNode deletes a node from the mock graph
func (m *MockGraph) DeleteNode(ctx context.Context, nodeType, nodeID string) error {
	key := nodeType + ":" + nodeID
	delete(m.nodes, key) // Always delete, even if not exists (compatible with registry tests)
	return nil
}

// QueryNodes queries nodes from the mock graph
func (m *MockGraph) QueryNodes(ctx context.Context, nodeType string, filters map[string]interface{}) ([]map[string]interface{}, error) {
	var results []map[string]interface{}
	for _, props := range m.nodes {
		if props["type"] == nodeType {
			// Simple filter matching with special handling for slices
			matches := true
			for k, v := range filters {
				propValue := props[k]
				if !compareValues(propValue, v) {
					matches = false
					break
				}
			}
			if matches {
				results = append(results, props)
			}
		}
	}
	return results, nil
}

// compareValues compares two values, handling slices specially
func compareValues(a, b interface{}) bool {
	// Handle slice comparisons for capabilities (contains logic)
	aSlice, aIsSlice := a.([]string)
	bSlice, bIsSlice := b.([]string)

	if aIsSlice && bIsSlice {
		// For capabilities: check if any requested capability is in agent's capabilities
		// a = agent capabilities (e.g., ["deploy", "rollback", "canary"])
		// b = requested capabilities (e.g., ["deploy"])
		for _, requested := range bSlice {
			for _, available := range aSlice {
				if requested == available {
					return true // Found a match
				}
			}
		}
		return false // No matches found
	}

	// Handle interface{} slice comparisons
	aISlice, aIsISlice := a.([]interface{})
	bISlice, bIsISlice := b.([]interface{})

	if aIsISlice && bIsISlice {
		// Convert to string slices and check
		aStrings := make([]string, len(aISlice))
		bStrings := make([]string, len(bISlice))

		for i, v := range aISlice {
			if str, ok := v.(string); ok {
				aStrings[i] = str
			}
		}
		for i, v := range bISlice {
			if str, ok := v.(string); ok {
				bStrings[i] = str
			}
		}

		return compareValues(aStrings, bStrings)
	}

	// Handle slice contains operations (for capability matching)
	if aIsSlice && !bIsSlice {
		// Check if b is contained in a
		bStr, ok := b.(string)
		if !ok {
			return false
		}
		for _, item := range aSlice {
			if item == bStr {
				return true
			}
		}
		return false
	}

	if aIsISlice && !bIsSlice {
		// Check if b is contained in a
		for _, item := range aISlice {
			if compareValues(item, b) {
				return true
			}
		}
		return false
	}

	// Default comparison
	return a == b
}

// GetStats returns mock statistics with realistic test data
func (m *MockGraph) GetStats() map[string]interface{} {
	nodesByType := m.getNodesByType()
	return map[string]interface{}{
		"implementation": "mock_graph_with_test_data",
		"total_nodes":    len(m.nodes),
		"nodes_by_type":  nodesByType,
		"capabilities":   []string{"deploy", "rollback", "canary", "security", "compliance", "audit", "monitoring", "optimization", "scaling"},
		"active_agents":  nodesByType["agent"],
		"workflows":      nodesByType["workflow"],
		"conversations":  nodesByType["conversation"],
	}
}

// Helper method to get nodes by type
func (m *MockGraph) getNodesByType() map[string]int {
	byType := make(map[string]int)
	for _, props := range m.nodes {
		if nodeType, ok := props["type"].(string); ok {
			byType[nodeType]++
		}
	}
	return byType
}

// Reset clears all data from the mock graph (useful for test cleanup)
func (m *MockGraph) Reset() {
	m.nodes = make(map[string]map[string]interface{})
}

// GetNodeCount returns the total number of nodes in the mock graph
func (m *MockGraph) GetNodeCount() int {
	return len(m.nodes)
}

// GetAllNodes returns all nodes in the mock graph (useful for debugging tests)
func (m *MockGraph) GetAllNodes() map[string]map[string]interface{} {
	// Return a copy to prevent external modification
	result := make(map[string]map[string]interface{})
	for k, v := range m.nodes {
		nodeCopy := make(map[string]interface{})
		for nk, nv := range v {
			nodeCopy[nk] = nv
		}
		result[k] = nodeCopy
	}
	return result
}

// Edge operations (minimal implementation for testing)
func (m *MockGraph) AddEdge(ctx context.Context, sourceType, sourceID, targetType, targetID, edgeType string, properties map[string]interface{}) error {
	// Simple edge storage for testing - not used by most tests
	return nil
}

func (m *MockGraph) GetEdges(ctx context.Context, nodeType, nodeID string) ([]map[string]interface{}, error) {
	// Return empty edges for testing
	return []map[string]interface{}{}, nil
}

func (m *MockGraph) UpdateEdge(ctx context.Context, sourceType, sourceID, targetType, targetID, edgeType string, properties map[string]interface{}) error {
	// Simple edge update for testing
	return nil
}

func (m *MockGraph) DeleteEdge(ctx context.Context, sourceType, sourceID, targetType, targetID, edgeType string) error {
	// Simple edge deletion for testing
	return nil
}

func (m *MockGraph) Close(ctx context.Context) error {
	// No-op close for testing
	return nil
}
