package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

// UI API Handlers for React Frontend
// These handlers provide the REST API endpoints that the React UI will consume

// GraphDataHandler returns an HTTP handler for graph data visualization
// Following clean architecture: HTTP layer delegates to service layer
func (w *ConversationAwareWebBFF) GraphDataHandler() http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(rw, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Extract conversation ID from URL path: /api/graph/conversation/{id}
		pathParts := strings.Split(r.URL.Path, "/")
		if len(pathParts) < 5 || pathParts[4] == "" {
			http.Error(rw, "Conversation ID required", http.StatusBadRequest)
			return
		}
		conversationID := pathParts[4]

		w.logger.Debug("Graph data request", "conversationID", conversationID)

		// Delegate to service layer (dependency injection)
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

// ExecutionPlanHandler returns an HTTP handler for execution plan data
// Following clean architecture: HTTP layer delegates to service layer
func (w *ConversationAwareWebBFF) ExecutionPlanHandler() http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(rw, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Extract plan ID from URL path: /api/execution-plan/{id}
		pathParts := strings.Split(r.URL.Path, "/")
		if len(pathParts) < 4 || pathParts[3] == "" {
			http.Error(rw, "Plan ID required", http.StatusBadRequest)
			return
		}
		planID := pathParts[3]

		w.logger.Debug("Execution plan request", "planID", planID)

		// Delegate to service layer
		uiService := NewUIAPIServiceWithGraph(w.conversationService, w.userService, w.graph)
		planData, err := uiService.GetExecutionPlan(r.Context(), planID)
		if err != nil {
			w.logger.Error("Failed to get execution plan", err)
			http.Error(rw, "Internal server error", http.StatusInternalServerError)
			return
		}

		rw.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(rw).Encode(planData); err != nil {
			w.logger.Error("Failed to encode execution plan data", err)
			http.Error(rw, "Internal server error", http.StatusInternalServerError)
			return
		}
	})
}

// ConversationHistoryHandler returns an HTTP handler for conversation history
// Following clean architecture: HTTP layer delegates to service layer
func (w *ConversationAwareWebBFF) ConversationHistoryHandler() http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(rw, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Extract session ID from URL path: /api/conversations/{sessionID}
		pathParts := strings.Split(r.URL.Path, "/")
		if len(pathParts) < 4 || pathParts[3] == "" {
			http.Error(rw, "Session ID required", http.StatusBadRequest)
			return
		}
		sessionID := pathParts[3]

		w.logger.Debug("Conversation history request", "sessionID", sessionID)

		// Delegate to service layer
		uiService := NewUIAPIServiceWithGraph(w.conversationService, w.userService, w.graph)
		historyData, err := uiService.GetConversationHistory(r.Context(), sessionID)
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

// EnhancedWebSocketHandler returns an enhanced WebSocket handler for typed real-time updates
func (w *ConversationAwareWebBFF) EnhancedWebSocketHandler() http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		// Upgrade connection to WebSocket
		conn, err := upgrader.Upgrade(rw, r, nil)
		if err != nil {
			w.logger.Error("Failed to upgrade to WebSocket", err)
			return
		}

		// Get or generate session ID
		sessionID := r.URL.Query().Get("session_id")
		if sessionID == "" {
			sessionID = "session-" + uuid.New().String()
		}

		w.logger.Info("Enhanced WebSocket connection established", "session_id", sessionID)

		// Create enhanced connection handler
		enhancedConn := NewEnhancedWebSocketConnection(conn, sessionID, w)
		
		// Start handling messages (this will block until connection closes)
		enhancedConn.Start(r.Context())
		
		w.logger.Info("Enhanced WebSocket connection closed", "session_id", sessionID)
	})
}

// AgentStatusHandler returns an HTTP handler for agent status information
// Following clean architecture: HTTP layer delegates to service layer
func (w *ConversationAwareWebBFF) AgentStatusHandler() http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(rw, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		w.logger.Debug("Agent status request")

		// Delegate to service layer
		uiService := NewUIAPIServiceWithGraph(w.conversationService, w.userService, w.graph)
		agentStatus, err := uiService.GetAgentStatus(r.Context())
		if err != nil {
			w.logger.Error("Failed to get agent status", err)
			http.Error(rw, "Internal server error", http.StatusInternalServerError)
			return
		}

		rw.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(rw).Encode(agentStatus); err != nil {
			w.logger.Error("Failed to encode agent status", err)
			http.Error(rw, "Internal server error", http.StatusInternalServerError)
			return
		}
	})
}

// CreateWebServer extends the base WebBFF CreateWebServer with UI API routes
func (w *ConversationAwareWebBFF) CreateWebServer(addr string) *http.Server {
	mux := http.NewServeMux()

	// Existing routes from base WebBFF
	mux.Handle("/api/chat", w.ChatHandler())
	mux.Handle("/ws", w.WebSocketHandler())

	// New UI API routes for React frontend
	mux.Handle("/api/graph/conversation/", w.GraphDataHandler())
	mux.Handle("/api/execution-plan/", w.ExecutionPlanHandler())
	mux.Handle("/api/conversations/", w.ConversationHistoryHandler())
	mux.Handle("/api/agents/status", w.AgentStatusHandler())
	mux.Handle("/ws/enhanced", w.EnhancedWebSocketHandler())

	// Health check
	mux.HandleFunc("/health", func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		fmt.Fprintf(rw, `{"status":"ok","service":"conversation-aware-web-bff"}`)
	})

	return &http.Server{
		Addr:    addr,
		Handler: mux,
	}
}
