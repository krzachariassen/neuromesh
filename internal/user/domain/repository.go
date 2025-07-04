package domain

import "context"

// UserRepository defines the interface for user persistence operations
type UserRepository interface {
	// Schema management
	EnsureUserSchema(ctx context.Context) error
	EnsureSessionSchema(ctx context.Context) error

	// User operations
	CreateUser(ctx context.Context, user *User) error
	GetUser(ctx context.Context, userID string) (*User, error)
	GetUserWithSessions(ctx context.Context, userID string) (*User, error)
	UpdateUser(ctx context.Context, user *User) error
	DeleteUser(ctx context.Context, userID string) error

	// Session operations
	CreateSession(ctx context.Context, session *Session) error
	GetSession(ctx context.Context, sessionID string) (*Session, error)
	GetUserSessions(ctx context.Context, userID string) ([]*Session, error)
	UpdateSession(ctx context.Context, session *Session) error
	DeleteSession(ctx context.Context, sessionID string) error

	// Relationship operations
	LinkUserToSession(ctx context.Context, userID, sessionID string) error
	UnlinkUserFromSession(ctx context.Context, userID, sessionID string) error

	// Query operations
	FindUsersByType(ctx context.Context, userType UserType) ([]*User, error)
	FindActiveUsers(ctx context.Context) ([]*User, error)
	FindExpiredSessions(ctx context.Context) ([]*Session, error)
}
