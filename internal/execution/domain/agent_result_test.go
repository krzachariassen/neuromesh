package domain

import (
	"testing"
	"time"
)

// RED Phase: Write failing tests that expose the design requirements for AgentResult domain entity
func TestAgentResult_Creation_ShouldCreateValidAgentResult(t *testing.T) {
	// Arrange
	stepID := "step-123"
	agentID := "agent-456"
	content := "Agent executed successfully with diagnostic results"
	metadata := map[string]interface{}{
		"execution_time": 2.5,
		"confidence":     0.95,
		"model_version":  "v1.2.3",
	}

	// Act
	result := NewAgentResult(stepID, agentID, content, metadata)

	// Assert
	if result == nil {
		t.Fatal("NewAgentResult should return a non-nil AgentResult")
	}

	if result.ID == "" {
		t.Error("AgentResult should have a generated ID")
	}

	if result.ExecutionStepID != stepID {
		t.Errorf("Expected ExecutionStepID %s, got %s", stepID, result.ExecutionStepID)
	}

	if result.AgentID != agentID {
		t.Errorf("Expected AgentID %s, got %s", agentID, result.AgentID)
	}

	if result.Content != content {
		t.Errorf("Expected Content %s, got %s", content, result.Content)
	}

	if result.Status != AgentResultStatusSuccess {
		t.Errorf("Expected default status Success, got %v", result.Status)
	}

	if result.Timestamp.IsZero() {
		t.Error("AgentResult should have a timestamp set")
	}

	if time.Since(result.Timestamp) > time.Second {
		t.Error("Timestamp should be recent")
	}

	if len(result.Metadata) != len(metadata) {
		t.Errorf("Expected metadata length %d, got %d", len(metadata), len(result.Metadata))
	}
}

func TestAgentResult_Creation_WithStatus_ShouldSetCorrectStatus(t *testing.T) {
	// Arrange
	stepID := "step-123"
	agentID := "agent-456"
	content := "Partial results due to timeout"
	status := AgentResultStatusPartial

	// Act
	result := NewAgentResultWithStatus(stepID, agentID, content, nil, status)

	// Assert
	if result.Status != status {
		t.Errorf("Expected status %v, got %v", status, result.Status)
	}
}

func TestAgentResult_Validation_ShouldValidateRequiredFields(t *testing.T) {
	tests := []struct {
		name        string
		result      *AgentResult
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid result",
			result: &AgentResult{
				ID:              "result-123",
				ExecutionStepID: "step-456",
				AgentID:         "agent-789",
				Content:         "Valid content",
				Status:          AgentResultStatusSuccess,
				Timestamp:       time.Now(),
				Metadata:        map[string]interface{}{},
			},
			expectError: false,
		},
		{
			name: "empty ID",
			result: &AgentResult{
				ExecutionStepID: "step-456",
				AgentID:         "agent-789",
				Content:         "Valid content",
				Status:          AgentResultStatusSuccess,
				Timestamp:       time.Now(),
			},
			expectError: true,
			errorMsg:    "ID is required",
		},
		{
			name: "empty ExecutionStepID",
			result: &AgentResult{
				ID:      "result-123",
				AgentID: "agent-789",
				Content: "Valid content",
				Status:  AgentResultStatusSuccess,
			},
			expectError: true,
			errorMsg:    "ExecutionStepID is required",
		},
		{
			name: "empty AgentID",
			result: &AgentResult{
				ID:              "result-123",
				ExecutionStepID: "step-456",
				Content:         "Valid content",
				Status:          AgentResultStatusSuccess,
			},
			expectError: true,
			errorMsg:    "AgentID is required",
		},
		{
			name: "invalid status",
			result: &AgentResult{
				ID:              "result-123",
				ExecutionStepID: "step-456",
				AgentID:         "agent-789",
				Content:         "Valid content",
				Status:          "invalid_status",
				Timestamp:       time.Now(),
			},
			expectError: true,
			errorMsg:    "invalid status: invalid_status",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Act
			err := test.result.Validate()

			// Assert
			if test.expectError {
				if err == nil {
					t.Errorf("Expected validation error, got nil")
				} else if test.errorMsg != "" && err.Error() != test.errorMsg {
					t.Errorf("Expected error message containing '%s', got '%s'", test.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no validation error, got: %v", err)
				}
			}
		})
	}
}

func TestAgentResult_StatusConstants_ShouldDefineCorrectValues(t *testing.T) {
	// Assert that status constants are defined
	if AgentResultStatusSuccess == "" {
		t.Error("AgentResultStatusSuccess should be defined")
	}

	if AgentResultStatusFailed == "" {
		t.Error("AgentResultStatusFailed should be defined")
	}

	if AgentResultStatusPartial == "" {
		t.Error("AgentResultStatusPartial should be defined")
	}

	// Status values should be meaningful
	expectedStatuses := []AgentResultStatus{
		AgentResultStatusSuccess,
		AgentResultStatusFailed,
		AgentResultStatusPartial,
	}

	for _, status := range expectedStatuses {
		if string(status) == "" {
			t.Errorf("Status %v should have a non-empty string value", status)
		}
	}
}

func TestAgentResult_IsSuccessful_ShouldReturnCorrectStatus(t *testing.T) {
	tests := []struct {
		status   AgentResultStatus
		expected bool
	}{
		{AgentResultStatusSuccess, true},
		{AgentResultStatusFailed, false},
		{AgentResultStatusPartial, false},
	}

	for _, test := range tests {
		t.Run(string(test.status), func(t *testing.T) {
			// Arrange
			result := &AgentResult{
				ID:              "test-id",
				ExecutionStepID: "step-id",
				AgentID:         "agent-id",
				Content:         "test content",
				Status:          test.status,
				Timestamp:       time.Now(),
				Metadata:        map[string]interface{}{},
			}

			// Act
			isSuccessful := result.IsSuccessful()

			// Assert
			if isSuccessful != test.expected {
				t.Errorf("Expected IsSuccessful() to return %v for status %v, got %v",
					test.expected, test.status, isSuccessful)
			}
		})
	}
}

func TestAgentResult_MarkAsFailed_ShouldUpdateStatusAndAddErrorInfo(t *testing.T) {
	// Arrange
	result := &AgentResult{
		ID:              "test-id",
		ExecutionStepID: "step-id",
		AgentID:         "agent-id",
		Content:         "initial content",
		Status:          AgentResultStatusSuccess,
		Timestamp:       time.Now(),
		Metadata:        map[string]interface{}{},
	}
	errorMsg := "Agent execution failed due to timeout"

	// Act
	result.MarkAsFailed(errorMsg)

	// Assert
	if result.Status != AgentResultStatusFailed {
		t.Errorf("Expected status to be Failed, got %v", result.Status)
	}

	errorInfo, exists := result.Metadata["error"]
	if !exists {
		t.Error("Expected error information in metadata")
	}

	if errorInfo != errorMsg {
		t.Errorf("Expected error message '%s', got '%v'", errorMsg, errorInfo)
	}
}
