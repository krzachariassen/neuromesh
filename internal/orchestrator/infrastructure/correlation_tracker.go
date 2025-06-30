package infrastructure

import (
	"context"
	"sync"
	"time"

	"neuromesh/internal/messaging"
)

// CorrelationRequest represents a pending request waiting for a response
type CorrelationRequest struct {
	CorrelationID string
	UserID        string
	ResponseChan  chan *messaging.AgentToAIMessage
	ExpiresAt     time.Time
}

// CorrelationTracker manages pending requests and routes responses by correlation ID
type CorrelationTracker struct {
	mu       sync.RWMutex
	requests map[string]*CorrelationRequest
}

// NewCorrelationTracker creates a new instance of CorrelationTracker
func NewCorrelationTracker() *CorrelationTracker {
	return &CorrelationTracker{
		requests: make(map[string]*CorrelationRequest),
	}
}

// RegisterRequest registers a new request with a correlation ID and returns a channel for the response
func (ct *CorrelationTracker) RegisterRequest(correlationID, userID string, timeout time.Duration) chan *messaging.AgentToAIMessage {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	responseChan := make(chan *messaging.AgentToAIMessage, 1)

	request := &CorrelationRequest{
		CorrelationID: correlationID,
		UserID:        userID,
		ResponseChan:  responseChan,
		ExpiresAt:     time.Now().Add(timeout),
	}

	ct.requests[correlationID] = request
	return responseChan
}

// RouteResponse routes an agent response to the appropriate waiting request
// Returns true if the response was routed successfully, false if no matching request was found
func (ct *CorrelationTracker) RouteResponse(response *messaging.AgentToAIMessage) bool {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	request, exists := ct.requests[response.CorrelationID]
	if !exists {
		return false
	}

	// Send response to waiting channel (non-blocking)
	select {
	case request.ResponseChan <- response:
		// Successfully sent, now clean up the request
		delete(ct.requests, response.CorrelationID)
		return true
	default:
		// Channel is full or closed, clean up anyway
		delete(ct.requests, response.CorrelationID)
		return false
	}
}

// CleanupRequest removes a pending request from the tracker
func (ct *CorrelationTracker) CleanupRequest(correlationID string) {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	if request, exists := ct.requests[correlationID]; exists {
		close(request.ResponseChan)
		delete(ct.requests, correlationID)
	}
}

// StartCleanupWorker starts a background worker that periodically cleans up expired requests
func (ct *CorrelationTracker) StartCleanupWorker(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(10 * time.Millisecond) // Frequent cleanup for testing
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				ct.cleanupExpiredRequests()
			}
		}
	}()
}

// cleanupExpiredRequests removes expired requests from the tracker
func (ct *CorrelationTracker) cleanupExpiredRequests() {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	now := time.Now()
	for correlationID, request := range ct.requests {
		if now.After(request.ExpiresAt) {
			close(request.ResponseChan)
			delete(ct.requests, correlationID)
		}
	}
}
