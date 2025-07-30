package controllers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	convDomain "neuromesh/internal/conversation/domain"
	"neuromesh/testHelpers"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConversationController_GetConversation(t *testing.T) {
	// Arrange
	mockConversationService := testHelpers.NewMockConversationService()
	controller := NewConversationController(mockConversationService)

	conversationID := "test-conversation-id"
	expectedConversation := &convDomain.Conversation{
		ID:     conversationID,
		Status: convDomain.ConversationStatusActive,
		// Note: SessionID and UserID are now handled via graph relationships, not entity properties
	}

	mockConversationService.On("GetConversation", context.Background(), conversationID).
		Return(expectedConversation, nil)

	// Act
	req := httptest.NewRequest(http.MethodGet, "/api/v1/conversations/"+conversationID, nil)
	w := httptest.NewRecorder()

	controller.GetConversation(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, conversationID, response["id"])
	assert.Equal(t, "test-session", response["session_id"])
	assert.Equal(t, "test-user", response["user_id"])
	assert.Equal(t, "active", response["status"])
}

func TestConversationController_GetConversationGraph(t *testing.T) {
	// Arrange
	mockConversationService := testHelpers.NewMockConversationService()
	mockGraphService := testHelpers.NewMockGraphService()
	controller := NewConversationController(mockConversationService)
	controller.SetGraphService(mockGraphService)

	conversationID := "test-conversation-id"
	expectedGraph := &convDomain.GraphData{
		Nodes: []convDomain.GraphNode{
			{ID: "node1", Type: "user", Data: map[string]interface{}{"name": "User Input"}},
			{ID: "node2", Type: "agent", Data: map[string]interface{}{"name": "Agent Response"}},
		},
		Edges: []convDomain.GraphEdge{
			{ID: "edge1", Source: "node1", Target: "node2", Type: "flow"},
		},
	}

	mockGraphService.On("GetConversationGraph", context.Background(), conversationID).
		Return(expectedGraph, nil)

	// Act
	req := httptest.NewRequest(http.MethodGet, "/api/v1/conversations/"+conversationID+"/graph", nil)
	w := httptest.NewRecorder()

	controller.GetConversationGraph(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response, "nodes")
	assert.Contains(t, response, "edges")

	nodes := response["nodes"].([]interface{})
	assert.Len(t, nodes, 2)

	edges := response["edges"].([]interface{})
	assert.Len(t, edges, 1)
}

func TestConversationController_InvalidConversationID(t *testing.T) {
	// Arrange
	mockConversationService := testHelpers.NewMockConversationService()
	controller := NewConversationController(mockConversationService)

	// Act - URL path without conversation ID
	req := httptest.NewRequest(http.MethodGet, "/api/v1/conversations/", nil)
	w := httptest.NewRecorder()

	controller.GetConversation(w, req)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, strings.ToLower(response["error"].(string)), "conversation id")
}

func TestConversationController_MethodNotAllowed(t *testing.T) {
	// Arrange
	mockConversationService := testHelpers.NewMockConversationService()
	controller := NewConversationController(mockConversationService)

	// Act
	req := httptest.NewRequest(http.MethodPost, "/api/v1/conversations/test-id", nil)
	w := httptest.NewRecorder()

	controller.GetConversation(w, req)

	// Assert
	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}
