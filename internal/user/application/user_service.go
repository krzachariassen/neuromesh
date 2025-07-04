package application

import (
	"context"
	"fmt"
	"time"

	"neuromesh/internal/user/domain"
)

// UserService defines the application service interface for user management
type UserService interface {
	// User management
	CreateUser(ctx context.Context, id, sessionID string, userType domain.UserType) (*domain.User, error)
	GetUser(ctx context.Context, userID string) (*domain.User, error)
	GetUserWithSessions(ctx context.Context, userID string) (*domain.User, error)
	UpdateUserStatus(ctx context.Context, userID string, status domain.UserStatus) error
	UpdateUserLastSeen(ctx context.Context, userID string) error
	SetUserMetadata(ctx context.Context, userID, key string, value interface{}) error
	DeleteUser(ctx context.Context, userID string) error

	// Session management
	CreateSession(ctx context.Context, id, userID string, duration time.Duration) (*domain.Session, error)
	GetSession(ctx context.Context, sessionID string) (*domain.Session, error)
	GetUserSessions(ctx context.Context, userID string) ([]*domain.Session, error)
	ExtendSession(ctx context.Context, sessionID string, duration time.Duration) error
	CloseSession(ctx context.Context, sessionID string) error
	CleanupExpiredSessions(ctx context.Context) error

	// Query operations
	FindUsersByType(ctx context.Context, userType domain.UserType) ([]*domain.User, error)
	FindActiveUsers(ctx context.Context) ([]*domain.User, error)

	// Schema management
	EnsureSchema(ctx context.Context) error
}

// UserServiceImpl implements the UserService interface
type UserServiceImpl struct {
	repo domain.UserRepository
}

// NewUserService creates a new user service implementation
func NewUserService(repo domain.UserRepository) UserService {
	return &UserServiceImpl{
		repo: repo,
	}
}

// CreateUser creates a new user
func (s *UserServiceImpl) CreateUser(ctx context.Context, id, sessionID string, userType domain.UserType) (*domain.User, error) {
	user, err := domain.NewUser(id, sessionID, userType)
	if err != nil {
		return nil, fmt.Errorf("failed to create user domain object: %w", err)
	}

	if err := s.repo.CreateUser(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to store user: %w", err)
	}

	return user, nil
}

// GetUser retrieves a user by ID
func (s *UserServiceImpl) GetUser(ctx context.Context, userID string) (*domain.User, error) {
	user, err := s.repo.GetUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return user, nil
}

// GetUserWithSessions retrieves a user with their sessions
func (s *UserServiceImpl) GetUserWithSessions(ctx context.Context, userID string) (*domain.User, error) {
	user, err := s.repo.GetUserWithSessions(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user with sessions: %w", err)
	}
	return user, nil
}

// UpdateUserStatus updates a user's status
func (s *UserServiceImpl) UpdateUserStatus(ctx context.Context, userID string, status domain.UserStatus) error {
	user, err := s.repo.GetUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	user.SetStatus(status)

	if err := s.repo.UpdateUser(ctx, user); err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

// UpdateUserLastSeen updates a user's last seen timestamp
func (s *UserServiceImpl) UpdateUserLastSeen(ctx context.Context, userID string) error {
	user, err := s.repo.GetUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	user.UpdateLastSeen()

	if err := s.repo.UpdateUser(ctx, user); err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

// SetUserMetadata sets metadata for a user
func (s *UserServiceImpl) SetUserMetadata(ctx context.Context, userID, key string, value interface{}) error {
	user, err := s.repo.GetUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	user.SetMetadata(key, value)

	if err := s.repo.UpdateUser(ctx, user); err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

// DeleteUser deletes a user
func (s *UserServiceImpl) DeleteUser(ctx context.Context, userID string) error {
	if err := s.repo.DeleteUser(ctx, userID); err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	return nil
}

// CreateSession creates a new session
func (s *UserServiceImpl) CreateSession(ctx context.Context, id, userID string, duration time.Duration) (*domain.Session, error) {
	session, err := domain.NewSession(id, userID, duration)
	if err != nil {
		return nil, fmt.Errorf("failed to create session domain object: %w", err)
	}

	if err := s.repo.CreateSession(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to store session: %w", err)
	}

	// Link user to session
	if err := s.repo.LinkUserToSession(ctx, userID, id); err != nil {
		return nil, fmt.Errorf("failed to link user to session: %w", err)
	}

	return session, nil
}

// GetSession retrieves a session by ID
func (s *UserServiceImpl) GetSession(ctx context.Context, sessionID string) (*domain.Session, error) {
	session, err := s.repo.GetSession(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}
	return session, nil
}

// GetUserSessions retrieves all sessions for a user
func (s *UserServiceImpl) GetUserSessions(ctx context.Context, userID string) ([]*domain.Session, error) {
	sessions, err := s.repo.GetUserSessions(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user sessions: %w", err)
	}
	return sessions, nil
}

// ExtendSession extends a session's expiration time
func (s *UserServiceImpl) ExtendSession(ctx context.Context, sessionID string, duration time.Duration) error {
	session, err := s.repo.GetSession(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	session.ExtendExpiration(duration)

	if err := s.repo.UpdateSession(ctx, session); err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}

	return nil
}

// CloseSession closes a session
func (s *UserServiceImpl) CloseSession(ctx context.Context, sessionID string) error {
	session, err := s.repo.GetSession(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	session.Close()

	if err := s.repo.UpdateSession(ctx, session); err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}

	return nil
}

// CleanupExpiredSessions marks expired sessions and optionally deletes them
func (s *UserServiceImpl) CleanupExpiredSessions(ctx context.Context) error {
	expiredSessions, err := s.repo.FindExpiredSessions(ctx)
	if err != nil {
		return fmt.Errorf("failed to find expired sessions: %w", err)
	}

	for _, session := range expiredSessions {
		if session.Status != domain.SessionStatusExpired {
			session.MarkExpired()
			if err := s.repo.UpdateSession(ctx, session); err != nil {
				return fmt.Errorf("failed to mark session as expired: %w", err)
			}
		}
	}

	return nil
}

// FindUsersByType finds users by type
func (s *UserServiceImpl) FindUsersByType(ctx context.Context, userType domain.UserType) ([]*domain.User, error) {
	users, err := s.repo.FindUsersByType(ctx, userType)
	if err != nil {
		return nil, fmt.Errorf("failed to find users by type: %w", err)
	}
	return users, nil
}

// FindActiveUsers finds all active users
func (s *UserServiceImpl) FindActiveUsers(ctx context.Context) ([]*domain.User, error) {
	users, err := s.repo.FindActiveUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find active users: %w", err)
	}
	return users, nil
}

// EnsureSchema ensures the user and session schemas are in place
func (s *UserServiceImpl) EnsureSchema(ctx context.Context) error {
	if err := s.repo.EnsureUserSchema(ctx); err != nil {
		return fmt.Errorf("failed to ensure user schema: %w", err)
	}

	if err := s.repo.EnsureSessionSchema(ctx); err != nil {
		return fmt.Errorf("failed to ensure session schema: %w", err)
	}

	return nil
}
