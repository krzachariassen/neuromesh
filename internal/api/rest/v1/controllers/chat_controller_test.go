package controllers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"neuromesh/internal/api/rest/v1/domain"
	"neuromesh/testHelpers"
)

func TestChatController_StartNewConversation_RequestValidation(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		expectedStatus int
		expectedError  string
	}{
		{
			name: "should_fail_with_empty_message",
			requestBody: domain.ChatRequest{
				Message:   "",
				ProjectID: "project-123",
				UserID:    "user-456",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "message is required",
		},
		{
			name: "should_fail_with_empty_project_id",
			requestBody: domain.ChatRequest{
				Message:   "Hello",
				ProjectID: "",
				UserID:    "user-456",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "project_id is required",
		},
		{
			name: "should_fail_with_empty_user_id",
			requestBody: domain.ChatRequest{
				Message:   "Hello",
				ProjectID: "project-123",
				UserID:    "",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "user_id is required",
		},
		{
			name:           "should_fail_with_invalid_json",
			requestBody:    `{"invalid": json}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid request body",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mocks - using nil for services since we're only testing validation
			convSvc := testHelpers.NewMockConversationService()
			userSvc := testHelpers.NewMockUserService()

			// Create controller - pass nil for project service since we're only testing validation
			controller := &ChatController{
				conversationService: convSvc,
				projectService:      nil, // Will skip project validation for these tests
				userService:         userSvc,
			}

			// Create request
			var reqBody []byte
			if str, ok := tt.requestBody.(string); ok {
				reqBody = []byte(str)
			} else {
				reqBody, _ = json.Marshal(tt.requestBody)
			}

			req := httptest.NewRequest(http.MethodPost, "/api/v1/chat", bytes.NewReader(reqBody))
			req.Header.Set("Content-Type", "application/json")

			// Create response recorder
			w := httptest.NewRecorder()

			// Call handler
			controller.StartNewConversation(w, req)

			// Assert status code
			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			// Assert error message if expected
			if tt.expectedError != "" {
				var errorResp domain.ErrorResponse
				json.NewDecoder(w.Body).Decode(&errorResp)
				if errorResp.Error != tt.expectedError {
					t.Errorf("expected error %q, got %q", tt.expectedError, errorResp.Error)
				}
			}
		})
	}
}

func TestChatController_StartNewConversation_MethodValidation(t *testing.T) {
	// Create mocks
	convSvc := testHelpers.NewMockConversationService()
	userSvc := testHelpers.NewMockUserService()

	// Create controller
	controller := &ChatController{
		conversationService: convSvc,
		projectService:      nil,
		userService:         userSvc,
	}

	// Test invalid HTTP method
	req := httptest.NewRequest(http.MethodGet, "/api/v1/chat", nil)
	w := httptest.NewRecorder()

	controller.StartNewConversation(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}
}

func TestChatController_ContinueConversation_RequestValidation(t *testing.T) {
	tests := []struct {
		name           string
		conversationID string
		requestBody    interface{}
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "should_fail_with_empty_message",
			conversationID: "conv-123",
			requestBody: domain.ChatMessage{
				Message: "",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "message is required",
		},
		{
			name:           "should_fail_with_empty_conversation_id",
			conversationID: "",
			requestBody: domain.ChatMessage{
				Message: "Hello",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "conversation_id is required",
		},
		{
			name:           "should_fail_with_invalid_json",
			conversationID: "conv-123",
			requestBody:    `{"invalid": json}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid request body",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mocks
			convSvc := testHelpers.NewMockConversationService()
			userSvc := testHelpers.NewMockUserService()

			// Create controller
			controller := &ChatController{
				conversationService: convSvc,
				projectService:      nil,
				userService:         userSvc,
			}

			// Create request
			var reqBody []byte
			if str, ok := tt.requestBody.(string); ok {
				reqBody = []byte(str)
			} else {
				reqBody, _ = json.Marshal(tt.requestBody)
			}

			url := "/api/v1/chat/" + tt.conversationID
			req := httptest.NewRequest(http.MethodPost, url, bytes.NewReader(reqBody))
			req.Header.Set("Content-Type", "application/json")

			// Create response recorder
			w := httptest.NewRecorder()

			// Call handler
			controller.ContinueConversation(w, req)

			// Assert status code
			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			// Assert error message if expected
			if tt.expectedError != "" {
				var errorResp domain.ErrorResponse
				json.NewDecoder(w.Body).Decode(&errorResp)
				if errorResp.Error != tt.expectedError {
					t.Errorf("expected error %q, got %q", tt.expectedError, errorResp.Error)
				}
			}
		})
	}
}
