package web

import (
	"encoding/json"
	"net/http"
	"strings"
)

// ConversationListHandler returns an HTTP handler for listing all conversations
// This replaces the old /api/graph/conversation/ with clean API abstraction
func (w *ConversationAwareWebBFF) ConversationListHandler() http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(rw, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		w.logger.Debug("Conversation list request")

		// Get all conversations through the service layer
		conversations, err := w.conversationService.GetAllConversations(r.Context())
		if err != nil {
			w.logger.Error("Failed to get all conversations", err)
			http.Error(rw, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Convert to response DTOs
		response := make([]ConversationSummary, len(conversations))
		for i, conv := range conversations {
			response[i] = ConversationSummary{
				ID:        conv.ID,
				SessionID: conv.SessionID,
				UserID:    conv.UserID,
				Status:    string(conv.Status),
				CreatedAt: conv.CreatedAt.Format("2006-01-02T15:04:05Z"),
				UpdatedAt: conv.UpdatedAt.Format("2006-01-02T15:04:05Z"),
			}
		}

		rw.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(rw).Encode(response); err != nil {
			w.logger.Error("Failed to encode conversation list", err)
			http.Error(rw, "Internal server error", http.StatusInternalServerError)
			return
		}
	})
}

// ConversationDetailHandler returns an HTTP handler for getting specific conversation details
// This replaces the old /api/graph/conversation/{id} with clean API abstraction
func (w *ConversationAwareWebBFF) ConversationDetailHandler() http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(rw, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Extract conversation ID from URL path: /api/conversation/{id}
		pathParts := strings.Split(r.URL.Path, "/")
		if len(pathParts) < 4 || pathParts[3] == "" {
			http.Error(rw, "Conversation ID required", http.StatusBadRequest)
			return
		}
		conversationID := pathParts[3]

		w.logger.Debug("Conversation detail request", "conversationID", conversationID)

		// Delegate to service layer (dependency injection)
		uiService := NewUIAPIServiceWithGraph(w.conversationService, w.userService, w.graph)
		graphData, err := uiService.GetGraphData(r.Context(), conversationID)
		if err != nil {
			w.logger.Error("Failed to get conversation data", err)
			http.Error(rw, "Internal server error", http.StatusInternalServerError)
			return
		}

		rw.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(rw).Encode(graphData); err != nil {
			w.logger.Error("Failed to encode conversation data", err)
			http.Error(rw, "Internal server error", http.StatusInternalServerError)
			return
		}
	})
}
