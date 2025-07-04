package domain

import (
	"fmt"
	"time"
)

// SessionStatus represents the status of a session
type SessionStatus string

const (
	SessionStatusActive  SessionStatus = "active"
	SessionStatusExpired SessionStatus = "expired"
	SessionStatusClosed  SessionStatus = "closed"
)

// Session represents a user session for tracking user activity
type Session struct {
	ID        string                 `json:"id"`
	UserID    string                 `json:"user_id"`
	Status    SessionStatus          `json:"status"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
	ExpiresAt time.Time              `json:"expires_at"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// NewSession creates a new session with validation
func NewSession(id, userID string, duration time.Duration) (*Session, error) {
	now := time.Now().UTC()

	session := &Session{
		ID:        id,
		UserID:    userID,
		Status:    SessionStatusActive,
		CreatedAt: now,
		UpdatedAt: now,
		ExpiresAt: now.Add(duration),
		Metadata:  make(map[string]interface{}),
	}

	if err := session.Validate(); err != nil {
		return nil, err
	}

	return session, nil
}

// Validate validates the session
func (s *Session) Validate() error {
	if s.ID == "" {
		return UserValidationError{Field: "id", Message: "ID cannot be empty"}
	}

	if s.UserID == "" {
		return UserValidationError{Field: "user_id", Message: "user ID cannot be empty"}
	}

	if s.CreatedAt.IsZero() {
		return UserValidationError{Field: "created_at", Message: "created_at cannot be zero"}
	}

	if s.ExpiresAt.Before(s.CreatedAt) {
		return UserValidationError{Field: "expires_at", Message: "expires_at cannot be before created_at"}
	}

	return nil
}

// IsExpired returns true if the session has expired
func (s *Session) IsExpired() bool {
	return time.Now().UTC().After(s.ExpiresAt)
}

// ExtendExpiration extends the session expiration time
func (s *Session) ExtendExpiration(duration time.Duration) {
	s.ExpiresAt = time.Now().UTC().Add(duration)
	s.UpdatedAt = time.Now().UTC()
}

// Close closes the session
func (s *Session) Close() {
	s.Status = SessionStatusClosed
	s.UpdatedAt = time.Now().UTC()
}

// MarkExpired marks the session as expired
func (s *Session) MarkExpired() {
	s.Status = SessionStatusExpired
	s.UpdatedAt = time.Now().UTC()
}

// String returns a string representation of the session
func (s *Session) String() string {
	return fmt.Sprintf("Session{ID: %s, UserID: %s, Status: %s, ExpiresAt: %s}",
		s.ID, s.UserID, s.Status, s.ExpiresAt.Format(time.RFC3339))
}
