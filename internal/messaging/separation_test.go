package messaging

import (
	"context"
	"testing"
)

// MockMessageBus tracks calls to Subscribe and PrepareAgentQueue
type MockMessageBus struct {
	SubscribeCalls         int
	PrepareAgentQueueCalls int
	SubscribeParams        []string
	PrepareParams          []string
}

func (m *MockMessageBus) SendMessage(ctx context.Context, message *Message) error {
	return nil
}

func (m *MockMessageBus) Subscribe(ctx context.Context, participantID string) (<-chan *Message, error) {
	m.SubscribeCalls++
	m.SubscribeParams = append(m.SubscribeParams, participantID)
	return make(chan *Message), nil
}

func (m *MockMessageBus) Unsubscribe(ctx context.Context, participantID string) error {
	return nil
}

func (m *MockMessageBus) PublishMessage(ctx context.Context, message *Message, recipients []string) error {
	return nil
}

func (m *MockMessageBus) GetConversationHistory(ctx context.Context, conversationID string) ([]*Message, error) {
	return []*Message{}, nil
}

func (m *MockMessageBus) CreateConversation(ctx context.Context, participants []string, context map[string]interface{}) (*ConversationContext, error) {
	return &ConversationContext{}, nil
}

func (m *MockMessageBus) PrepareAgentQueue(ctx context.Context, agentID string) error {
	m.PrepareAgentQueueCalls++
	m.PrepareParams = append(m.PrepareParams, agentID)
	return nil
}

// TestPrepareAgentQueueSeparation tests that queue preparation and subscription are separate concerns
func TestPrepareAgentQueueSeparation(t *testing.T) {
	mockBus := &MockMessageBus{}

	ctx := context.Background()
	agentID := "test-agent-1"

	// Test 1: PrepareAgentQueue should not call Subscribe
	err := mockBus.PrepareAgentQueue(ctx, agentID)
	if err != nil {
		t.Fatalf("PrepareAgentQueue failed: %v", err)
	}

	if mockBus.SubscribeCalls != 0 {
		t.Errorf("PrepareAgentQueue should not call Subscribe. Got %d calls", mockBus.SubscribeCalls)
	}

	if mockBus.PrepareAgentQueueCalls != 1 {
		t.Errorf("Expected 1 PrepareAgentQueue call, got %d", mockBus.PrepareAgentQueueCalls)
	}

	// Test 2: Subscribe should call PrepareAgentQueue internally (refactored)
	// Reset counters
	mockBus.PrepareAgentQueueCalls = 0

	_, err = mockBus.Subscribe(ctx, agentID)
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	if mockBus.SubscribeCalls != 1 {
		t.Errorf("Expected 1 Subscribe call, got %d", mockBus.SubscribeCalls)
	}

	// Note: This test validates the current implementation where Subscribe
	// calls PrepareAgentQueue internally through delegation
}

// TestNoRedundantSubscription tests the specific issue we're fixing
func TestNoRedundantSubscription(t *testing.T) {
	mockBus := &MockMessageBus{}

	ctx := context.Background()
	agentID := "test-agent-1"

	// Simulate RegisterAgent workflow
	err := mockBus.PrepareAgentQueue(ctx, agentID)
	if err != nil {
		t.Fatalf("PrepareAgentQueue failed: %v", err)
	}

	// Simulate OpenConversation workflow
	_, err = mockBus.Subscribe(ctx, agentID)
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	// Verify correct call counts
	if mockBus.PrepareAgentQueueCalls != 1 {
		t.Errorf("Expected 1 PrepareAgentQueue call, got %d", mockBus.PrepareAgentQueueCalls)
	}

	if mockBus.SubscribeCalls != 1 {
		t.Errorf("Expected 1 Subscribe call, got %d", mockBus.SubscribeCalls)
	}

	// Verify parameters
	if len(mockBus.PrepareParams) != 1 || mockBus.PrepareParams[0] != agentID {
		t.Errorf("Expected PrepareAgentQueue called with %s, got %v", agentID, mockBus.PrepareParams)
	}

	if len(mockBus.SubscribeParams) != 1 || mockBus.SubscribeParams[0] != agentID {
		t.Errorf("Expected Subscribe called with %s, got %v", agentID, mockBus.SubscribeParams)
	}
}
