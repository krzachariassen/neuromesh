package infrastructure

import (
	"context"
	"fmt"
	"time"

	"neuromesh/internal/graph"
	"neuromesh/internal/user/domain"
)

// Constants for graph node types and relationships
const (
	NodeTypeUser    = "User"
	NodeTypeSession = "Session"

	RelationshipHasSession = "HAS_SESSION"

	TimeFormat = "2006-01-02T15:04:05Z"
)

// GraphUserRepository implements user repository using the graph backend
type GraphUserRepository struct {
	graph graph.Graph
}

// NewGraphUserRepository creates a new graph-based user repository
func NewGraphUserRepository(g graph.Graph) domain.UserRepository {
	return &GraphUserRepository{
		graph: g,
	}
}

// formatTime formats time for graph storage
func formatTime(t time.Time) string {
	return t.Format(TimeFormat)
}

// parseTime parses time from graph storage
func parseTime(timeStr string) (time.Time, error) {
	return time.Parse(TimeFormat, timeStr)
}

// EnsureUserSchema ensures that the required schema for User domain is in place
func (r *GraphUserRepository) EnsureUserSchema(ctx context.Context) error {
	// Create unique constraints for User nodes
	if err := r.graph.CreateUniqueConstraint(ctx, NodeTypeUser, "id"); err != nil {
		return fmt.Errorf("failed to create user id constraint: %w", err)
	}

	if err := r.graph.CreateUniqueConstraint(ctx, NodeTypeUser, "session_id"); err != nil {
		return fmt.Errorf("failed to create user session_id constraint: %w", err)
	}

	// Create indexes for User nodes
	userIndexes := []string{"user_type", "status", "created_at", "last_seen"}
	for _, property := range userIndexes {
		if err := r.graph.CreateIndex(ctx, NodeTypeUser, property); err != nil {
			return fmt.Errorf("failed to create user %s index: %w", property, err)
		}
	}

	return nil
}

// EnsureSessionSchema ensures that the required schema for Session domain is in place
func (r *GraphUserRepository) EnsureSessionSchema(ctx context.Context) error {
	// Create unique constraint for Session nodes
	if err := r.graph.CreateUniqueConstraint(ctx, NodeTypeSession, "id"); err != nil {
		return fmt.Errorf("failed to create session id constraint: %w", err)
	}

	// Create indexes for Session nodes
	sessionIndexes := []string{"user_id", "status", "created_at", "expires_at"}
	for _, property := range sessionIndexes {
		if err := r.graph.CreateIndex(ctx, NodeTypeSession, property); err != nil {
			return fmt.Errorf("failed to create session %s index: %w", property, err)
		}
	}

	return nil
}

// CreateUser creates a user node in the graph
func (r *GraphUserRepository) CreateUser(ctx context.Context, user *domain.User) error {
	properties := map[string]interface{}{
		"id":         user.ID,
		"session_id": user.SessionID,
		"user_type":  string(user.UserType),
		"status":     string(user.Status),
		"created_at": formatTime(user.CreatedAt),
		"updated_at": formatTime(user.UpdatedAt),
		"last_seen":  formatTime(user.LastSeen),
	}

	// Add metadata if present
	if len(user.Metadata) > 0 {
		properties["metadata"] = user.Metadata
	}

	return r.graph.AddNode(ctx, NodeTypeUser, user.ID, properties)
}

// GetUser retrieves a user by ID
func (r *GraphUserRepository) GetUser(ctx context.Context, userID string) (*domain.User, error) {
	userProps, err := r.graph.GetNode(ctx, NodeTypeUser, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if userProps == nil {
		return nil, fmt.Errorf("user not found: %s", userID)
	}

	return r.mapToUser(userProps)
}

// GetUserWithSessions retrieves a user with their sessions
func (r *GraphUserRepository) GetUserWithSessions(ctx context.Context, userID string) (*domain.User, error) {
	// Get the user node
	userProps, err := r.graph.GetNode(ctx, NodeTypeUser, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if userProps == nil {
		return nil, fmt.Errorf("user not found: %s", userID)
	}

	// Convert map properties back to User domain object
	user, err := r.mapToUser(userProps)
	if err != nil {
		return nil, fmt.Errorf("failed to map user properties: %w", err)
	}

	return user, nil
}

// UpdateUser updates a user node in the graph
func (r *GraphUserRepository) UpdateUser(ctx context.Context, user *domain.User) error {
	properties := map[string]interface{}{
		"session_id": user.SessionID,
		"user_type":  string(user.UserType),
		"status":     string(user.Status),
		"updated_at": formatTime(user.UpdatedAt),
		"last_seen":  formatTime(user.LastSeen),
	}

	// Add metadata if present
	if len(user.Metadata) > 0 {
		properties["metadata"] = user.Metadata
	}

	return r.graph.UpdateNode(ctx, NodeTypeUser, user.ID, properties)
}

// DeleteUser deletes a user node from the graph
func (r *GraphUserRepository) DeleteUser(ctx context.Context, userID string) error {
	return r.graph.DeleteNode(ctx, NodeTypeUser, userID)
}

// CreateSession creates a session node in the graph
func (r *GraphUserRepository) CreateSession(ctx context.Context, session *domain.Session) error {
	properties := map[string]interface{}{
		"id":         session.ID,
		"user_id":    session.UserID,
		"status":     string(session.Status),
		"created_at": formatTime(session.CreatedAt),
		"updated_at": formatTime(session.UpdatedAt),
		"expires_at": formatTime(session.ExpiresAt),
	}

	// Add metadata if present
	if len(session.Metadata) > 0 {
		properties["metadata"] = session.Metadata
	}

	return r.graph.AddNode(ctx, NodeTypeSession, session.ID, properties)
}

// GetSession retrieves a session by ID
func (r *GraphUserRepository) GetSession(ctx context.Context, sessionID string) (*domain.Session, error) {
	sessionProps, err := r.graph.GetNode(ctx, NodeTypeSession, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	if sessionProps == nil {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	return r.mapToSession(sessionProps)
}

// GetUserSessions retrieves all sessions for a user
func (r *GraphUserRepository) GetUserSessions(ctx context.Context, userID string) ([]*domain.Session, error) {
	// Query sessions by user_id
	filters := map[string]interface{}{
		"user_id": userID,
	}

	sessionProps, err := r.graph.QueryNodes(ctx, NodeTypeSession, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to query user sessions: %w", err)
	}

	sessions := make([]*domain.Session, len(sessionProps))
	for i, props := range sessionProps {
		session, err := r.mapToSession(props)
		if err != nil {
			return nil, fmt.Errorf("failed to map session properties: %w", err)
		}
		sessions[i] = session
	}

	return sessions, nil
}

// UpdateSession updates a session node in the graph
func (r *GraphUserRepository) UpdateSession(ctx context.Context, session *domain.Session) error {
	properties := map[string]interface{}{
		"user_id":    session.UserID,
		"status":     string(session.Status),
		"updated_at": formatTime(session.UpdatedAt),
		"expires_at": formatTime(session.ExpiresAt),
	}

	// Add metadata if present
	if len(session.Metadata) > 0 {
		properties["metadata"] = session.Metadata
	}

	return r.graph.UpdateNode(ctx, NodeTypeSession, session.ID, properties)
}

// DeleteSession deletes a session node from the graph
func (r *GraphUserRepository) DeleteSession(ctx context.Context, sessionID string) error {
	return r.graph.DeleteNode(ctx, NodeTypeSession, sessionID)
}

// LinkUserToSession creates a relationship between user and session
func (r *GraphUserRepository) LinkUserToSession(ctx context.Context, userID, sessionID string) error {
	properties := map[string]interface{}{
		"created_at": formatTime(time.Now().UTC()),
	}

	return r.graph.AddEdge(ctx, NodeTypeUser, userID, NodeTypeSession, sessionID, RelationshipHasSession, properties)
}

// UnlinkUserFromSession removes the relationship between user and session
func (r *GraphUserRepository) UnlinkUserFromSession(ctx context.Context, userID, sessionID string) error {
	return r.graph.DeleteEdge(ctx, NodeTypeUser, userID, NodeTypeSession, sessionID, RelationshipHasSession)
}

// FindUsersByType finds users by type
func (r *GraphUserRepository) FindUsersByType(ctx context.Context, userType domain.UserType) ([]*domain.User, error) {
	filters := map[string]interface{}{
		"user_type": string(userType),
	}

	userProps, err := r.graph.QueryNodes(ctx, NodeTypeUser, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to query users by type: %w", err)
	}

	users := make([]*domain.User, len(userProps))
	for i, props := range userProps {
		user, err := r.mapToUser(props)
		if err != nil {
			return nil, fmt.Errorf("failed to map user properties: %w", err)
		}
		users[i] = user
	}

	return users, nil
}

// FindActiveUsers finds all active users
func (r *GraphUserRepository) FindActiveUsers(ctx context.Context) ([]*domain.User, error) {
	filters := map[string]interface{}{
		"status": string(domain.UserStatusActive),
	}

	userProps, err := r.graph.QueryNodes(ctx, NodeTypeUser, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to query active users: %w", err)
	}

	users := make([]*domain.User, len(userProps))
	for i, props := range userProps {
		user, err := r.mapToUser(props)
		if err != nil {
			return nil, fmt.Errorf("failed to map user properties: %w", err)
		}
		users[i] = user
	}

	return users, nil
}

// FindExpiredSessions finds all expired sessions
func (r *GraphUserRepository) FindExpiredSessions(ctx context.Context) ([]*domain.Session, error) {
	// This is a simplified implementation - in a real scenario, you'd use a more complex query
	// to find sessions where expires_at < current time
	filters := map[string]interface{}{} // Query all sessions, then filter

	sessionProps, err := r.graph.QueryNodes(ctx, NodeTypeSession, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to query sessions: %w", err)
	}

	var expiredSessions []*domain.Session
	for _, props := range sessionProps {
		session, err := r.mapToSession(props)
		if err != nil {
			return nil, fmt.Errorf("failed to map session properties: %w", err)
		}

		if session.IsExpired() {
			expiredSessions = append(expiredSessions, session)
		}
	}

	return expiredSessions, nil
}

// mapToUser converts map properties to User domain object
func (r *GraphUserRepository) mapToUser(props map[string]interface{}) (*domain.User, error) {
	id, ok := props["id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid user id")
	}

	sessionID, ok := props["session_id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid session_id")
	}

	userTypeStr, ok := props["user_type"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid user_type")
	}

	statusStr, ok := props["status"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid status")
	}

	createdAtStr, ok := props["created_at"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid created_at")
	}

	updatedAtStr, ok := props["updated_at"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid updated_at")
	}

	lastSeenStr, ok := props["last_seen"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid last_seen")
	}

	// Parse timestamps
	createdAt, err := parseTime(createdAtStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse created_at: %w", err)
	}

	updatedAt, err := parseTime(updatedAtStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse updated_at: %w", err)
	}

	lastSeen, err := parseTime(lastSeenStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse last_seen: %w", err)
	}

	// Create user object
	user := &domain.User{
		ID:        id,
		SessionID: sessionID,
		UserType:  domain.UserType(userTypeStr),
		Status:    domain.UserStatus(statusStr),
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
		LastSeen:  lastSeen,
		Metadata:  make(map[string]interface{}),
	}

	// Add metadata if present
	if metadata, exists := props["metadata"]; exists {
		if metadataMap, ok := metadata.(map[string]interface{}); ok {
			user.Metadata = metadataMap
		}
	}

	return user, nil
}

// mapToSession converts map properties to Session domain object
func (r *GraphUserRepository) mapToSession(props map[string]interface{}) (*domain.Session, error) {
	id, ok := props["id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid session id")
	}

	userID, ok := props["user_id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid user_id")
	}

	statusStr, ok := props["status"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid status")
	}

	createdAtStr, ok := props["created_at"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid created_at")
	}

	updatedAtStr, ok := props["updated_at"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid updated_at")
	}

	expiresAtStr, ok := props["expires_at"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid expires_at")
	}

	// Parse timestamps
	createdAt, err := parseTime(createdAtStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse created_at: %w", err)
	}

	updatedAt, err := parseTime(updatedAtStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse updated_at: %w", err)
	}

	expiresAt, err := parseTime(expiresAtStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse expires_at: %w", err)
	}

	// Create session object
	session := &domain.Session{
		ID:        id,
		UserID:    userID,
		Status:    domain.SessionStatus(statusStr),
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
		ExpiresAt: expiresAt,
		Metadata:  make(map[string]interface{}),
	}

	// Add metadata if present
	if metadata, exists := props["metadata"]; exists {
		if metadataMap, ok := metadata.(map[string]interface{}); ok {
			session.Metadata = metadataMap
		}
	}

	return session, nil
}
