package web

import (
	"encoding/json"
	"net/http"
)

// UIGraphDataHandler returns an HTTP handler for UI graph data requests
// This handles /api/ui/graph-data?conversation_id={id}
func (w *ConversationAwareWebBFF) UIGraphDataHandler() http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(rw, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Get conversation_id from query parameter
		conversationID := r.URL.Query().Get("conversation_id")
		if conversationID == "" {
			http.Error(rw, "conversation_id parameter is required", http.StatusBadRequest)
			return
		}

		w.logger.Debug("UI graph data request", "conversationID", conversationID)

		// Delegate to service layer
		uiService := NewUIAPIServiceWithGraph(w.conversationService, w.userService, w.graph)
		graphData, err := uiService.GetGraphData(r.Context(), conversationID)
		if err != nil {
			w.logger.Error("Failed to get graph data", err)
			http.Error(rw, "Internal server error", http.StatusInternalServerError)
			return
		}

		rw.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(rw).Encode(graphData); err != nil {
			w.logger.Error("Failed to encode graph data", err)
			http.Error(rw, "Internal server error", http.StatusInternalServerError)
			return
		}
	})
}

// UIConversationHistoryHandler returns an HTTP handler for UI conversation history requests
// This handles /api/ui/conversation-history?conversation_id={id}
func (w *ConversationAwareWebBFF) UIConversationHistoryHandler() http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(rw, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Get conversation_id from query parameter
		conversationID := r.URL.Query().Get("conversation_id")
		if conversationID == "" {
			http.Error(rw, "conversation_id parameter is required", http.StatusBadRequest)
			return
		}

		w.logger.Debug("UI conversation history request", "conversationID", conversationID)

		// Delegate to service layer
		uiService := NewUIAPIServiceWithGraph(w.conversationService, w.userService, w.graph)
		historyData, err := uiService.GetConversationHistory(r.Context(), conversationID)
		if err != nil {
			w.logger.Error("Failed to get conversation history", err)
			http.Error(rw, "Internal server error", http.StatusInternalServerError)
			return
		}

		rw.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(rw).Encode(historyData); err != nil {
			w.logger.Error("Failed to encode conversation history", err)
			http.Error(rw, "Internal server error", http.StatusInternalServerError)
			return
		}
	})
}

// UIExecutionPlansHandler returns an HTTP handler for UI execution plans requests
// This handles /api/ui/execution-plans?conversation_id={id}
func (w *ConversationAwareWebBFF) UIExecutionPlansHandler() http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(rw, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Get conversation_id from query parameter
		conversationID := r.URL.Query().Get("conversation_id")
		if conversationID == "" {
			http.Error(rw, "conversation_id parameter is required", http.StatusBadRequest)
			return
		}

		w.logger.Debug("UI execution plans request", "conversationID", conversationID)

		// Delegate to service layer
		uiService := NewUIAPIServiceWithGraph(w.conversationService, w.userService, w.graph)
		planData, err := uiService.GetExecutionPlan(r.Context(), conversationID)
		if err != nil {
			w.logger.Error("Failed to get execution plans", err)
			http.Error(rw, "Internal server error", http.StatusInternalServerError)
			return
		}

		rw.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(rw).Encode(planData); err != nil {
			w.logger.Error("Failed to encode execution plans", err)
			http.Error(rw, "Internal server error", http.StatusInternalServerError)
			return
		}
	})
}
