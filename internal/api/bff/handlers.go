package bff

import (
	"encoding/json"
	"net/http"
)

// ChatHandler handles HTTP requests for chat API endpoints
func (s *Service) ChatHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req ChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.logger.Error("Failed to decode chat request", err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Validate request
		if req.SessionID == "" {
			http.Error(w, "session_id is required", http.StatusBadRequest)
			return
		}

		if req.Message == "" {
			http.Error(w, "message is required", http.StatusBadRequest)
			return
		}

		// Process the message
		response, err := s.ProcessMessage(r.Context(), req.SessionID, req.Message)
		if err != nil {
			s.logger.Error("Failed to process web message", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Return JSON response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			s.logger.Error("Failed to encode response", err)
		}
	})
}

// WebSocketHandler handles WebSocket connections for real-time chat
func (s *Service) WebSocketHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get session ID from query parameter
		sessionID := r.URL.Query().Get("session_id")
		if sessionID == "" {
			http.Error(w, "session_id query parameter is required", http.StatusBadRequest)
			return
		}

		// Upgrade connection to WebSocket
		conn, err := s.upgrader.Upgrade(w, r, nil)
		if err != nil {
			s.logger.Error("Failed to upgrade WebSocket connection", err)
			return
		}
		defer conn.Close()

		s.logger.Info("WebSocket connection established", "sessionID", sessionID)

		// Handle WebSocket messages
		for {
			var msg struct {
				Type    string `json:"type"`
				Message string `json:"message"`
			}

			// Read message from WebSocket
			if err := conn.ReadJSON(&msg); err != nil {
				s.logger.Error("Failed to read WebSocket message", err)
				break
			}

			// Handle different message types
			switch msg.Type {
			case "chat":
				// Process chat message
				response, err := s.ProcessMessage(r.Context(), sessionID, msg.Message)
				if err != nil {
					s.logger.Error("Failed to process WebSocket message", err)
					// Send error response
					errorResponse := map[string]interface{}{
						"type":    "error",
						"message": "Failed to process message",
					}
					conn.WriteJSON(errorResponse)
					continue
				}

				// Send response back through WebSocket
				wsResponse := map[string]interface{}{
					"type":           "response",
					"content":        response.Content,
					"session_id":     response.SessionID,
					"correlation_id": response.CorrelationID,
				}
				if err := conn.WriteJSON(wsResponse); err != nil {
					s.logger.Error("Failed to send WebSocket response", err)
					break
				}

			case "ping":
				// Respond to ping with pong
				pongResponse := map[string]interface{}{
					"type": "pong",
				}
				if err := conn.WriteJSON(pongResponse); err != nil {
					s.logger.Error("Failed to send pong response", err)
					break
				}

			default:
				s.logger.Warn("Unknown WebSocket message type", "type", msg.Type)
			}
		}

		s.logger.Info("WebSocket connection closed", "sessionID", sessionID)
	})
}
