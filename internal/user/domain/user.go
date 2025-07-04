package domain

import (
	"fmt"
	"time"
)

// UserValidationError represents validation errors for users
type UserValidationError struct {
	Field   string
	Message string
}

func (e UserValidationError) Error() string {
	return fmt.Sprintf("user validation error - %s: %s", e.Field, e.Message)
}

// UserType represents the type of user session
type UserType string

const (
	UserTypeWebSession UserType = "web_session"
	UserTypeAPIUser    UserType = "api_user"
	UserTypeAgent      UserType = "agent"
	UserTypeSystem     UserType = "system"
)

// UserStatus represents the status of a user session
type UserStatus string

const (
	UserStatusActive   UserStatus = "active"
	UserStatusInactive UserStatus = "inactive"
	UserStatusBlocked  UserStatus = "blocked"
)

// User represents a user in the system for session and context tracking
type User struct {
	ID        string                 `json:"id"`
	SessionID string                 `json:"session_id"`
	UserType  UserType               `json:"user_type"`
	Status    UserStatus             `json:"status"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
	LastSeen  time.Time              `json:"last_seen"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// NewUser creates a new user with validation
func NewUser(id, sessionID string, userType UserType) (*User, error) {
	now := time.Now().UTC()

	user := &User{
		ID:        id,
		SessionID: sessionID,
		UserType:  userType,
		Status:    UserStatusActive,
		CreatedAt: now,
		UpdatedAt: now,
		LastSeen:  now,
		Metadata:  make(map[string]interface{}),
	}

	if err := user.Validate(); err != nil {
		return nil, err
	}

	return user, nil
}

// Validate validates the user
func (u *User) Validate() error {
	if u.ID == "" {
		return UserValidationError{Field: "id", Message: "ID cannot be empty"}
	}

	if u.SessionID == "" {
		return UserValidationError{Field: "session_id", Message: "session ID cannot be empty"}
	}

	if u.UserType == "" {
		return UserValidationError{Field: "user_type", Message: "user type cannot be empty"}
	}

	return nil
}

// UpdateLastSeen updates the last seen timestamp
func (u *User) UpdateLastSeen() {
	u.LastSeen = time.Now().UTC()
	u.UpdatedAt = time.Now().UTC()
}

// SetStatus updates the user status
func (u *User) SetStatus(status UserStatus) {
	u.Status = status
	u.UpdatedAt = time.Now().UTC()
}

// SetMetadata sets or updates metadata
func (u *User) SetMetadata(key string, value interface{}) {
	if u.Metadata == nil {
		u.Metadata = make(map[string]interface{})
	}
	u.Metadata[key] = value
	u.UpdatedAt = time.Now().UTC()
}

// GetMetadata retrieves metadata
func (u *User) GetMetadata(key string) (interface{}, bool) {
	if u.Metadata == nil {
		return nil, false
	}
	value, exists := u.Metadata[key]
	return value, exists
}
