package infrastructure

import (
	"context"
	"fmt"
	"time"

	"neuromesh/internal/graph"
	"neuromesh/internal/project/domain"
)

// Constants for graph node types and relationships
const (
	NodeTypeProject       = "Project"
	NodeTypeProjectMember = "ProjectMember"

	RelationshipHasMember = "HAS_MEMBER"

	TimeFormat = "2006-01-02T15:04:05Z"
)

// GraphProjectRepository implements ProjectRepository using the graph backend
type GraphProjectRepository struct {
	graph graph.Graph
}

// NewGraphProjectRepository creates a new graph-based project repository
func NewGraphProjectRepository(g graph.Graph) domain.ProjectRepository {
	return &GraphProjectRepository{
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

// EnsureProjectSchema ensures the project schema is in place
func (r *GraphProjectRepository) EnsureProjectSchema(ctx context.Context) error {
	// Create unique constraints for Project nodes
	if err := r.graph.CreateUniqueConstraint(ctx, NodeTypeProject, "id"); err != nil {
		return fmt.Errorf("failed to create project id constraint: %w", err)
	}

	// Create indexes for Project nodes
	projectIndexes := []string{"owner_email", "status", "created_at", "updated_at"}
	for _, property := range projectIndexes {
		if err := r.graph.CreateIndex(ctx, NodeTypeProject, property); err != nil {
			return fmt.Errorf("failed to create project %s index: %w", property, err)
		}
	}

	// Create indexes for ProjectMember nodes
	memberIndexes := []string{"user_id", "email", "role", "added_at"}
	for _, property := range memberIndexes {
		if err := r.graph.CreateIndex(ctx, NodeTypeProjectMember, property); err != nil {
			return fmt.Errorf("failed to create project member %s index: %w", property, err)
		}
	}

	return nil
}

// CreateProject creates a new project in the graph
func (r *GraphProjectRepository) CreateProject(ctx context.Context, project *domain.Project) error {
	properties := map[string]interface{}{
		"id":          project.ID,
		"name":        project.Name,
		"description": project.Description,
		"owner_email": project.OwnerEmail,
		"status":      string(project.Status),
		"created_at":  formatTime(project.CreatedAt),
		"updated_at":  formatTime(project.UpdatedAt),
	}

	return r.graph.AddNode(ctx, NodeTypeProject, project.ID, properties)
}

// GetProject retrieves a project by ID
func (r *GraphProjectRepository) GetProject(ctx context.Context, projectID string) (*domain.Project, error) {
	projectProps, err := r.graph.GetNode(ctx, NodeTypeProject, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	if projectProps == nil {
		return nil, fmt.Errorf("project not found: %s", projectID)
	}

	return r.mapToProject(projectProps)
}

// GetProjectWithMembers retrieves a project with all its members
func (r *GraphProjectRepository) GetProjectWithMembers(ctx context.Context, projectID string) (*domain.Project, error) {
	// Get the project first
	project, err := r.GetProject(ctx, projectID)
	if err != nil {
		return nil, err
	}

	// Get project members
	members, err := r.GetProjectMembers(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get project members: %w", err)
	}

	project.Members = members
	return project, nil
}

// UpdateProject updates an existing project
func (r *GraphProjectRepository) UpdateProject(ctx context.Context, project *domain.Project) error {
	properties := map[string]interface{}{
		"name":        project.Name,
		"description": project.Description,
		"status":      string(project.Status),
		"updated_at":  formatTime(project.UpdatedAt),
	}

	return r.graph.UpdateNode(ctx, NodeTypeProject, project.ID, properties)
}

// DeleteProject deletes a project and all its relationships
func (r *GraphProjectRepository) DeleteProject(ctx context.Context, projectID string) error {
	return r.graph.DeleteNode(ctx, NodeTypeProject, projectID)
}

// AddMember adds a member to a project
func (r *GraphProjectRepository) AddMember(ctx context.Context, projectID string, member *domain.ProjectMember) error {
	// Create the member node with a composite ID
	memberID := fmt.Sprintf("%s_%s", projectID, member.UserID)
	properties := map[string]interface{}{
		"user_id":  member.UserID,
		"email":    member.Email,
		"role":     string(member.Role),
		"added_at": formatTime(time.Now().UTC()),
	}

	if err := r.graph.AddNode(ctx, NodeTypeProjectMember, memberID, properties); err != nil {
		return fmt.Errorf("failed to create member node: %w", err)
	}

	// Create relationship between project and member
	relationshipProps := map[string]interface{}{
		"created_at": formatTime(time.Now().UTC()),
	}

	return r.graph.AddEdge(ctx, NodeTypeProject, projectID, NodeTypeProjectMember, memberID, RelationshipHasMember, relationshipProps)
}

// RemoveMember removes a member from a project
func (r *GraphProjectRepository) RemoveMember(ctx context.Context, projectID, userID string) error {
	memberID := fmt.Sprintf("%s_%s", projectID, userID)
	return r.graph.DeleteNode(ctx, NodeTypeProjectMember, memberID)
}

// GetProjectMembers retrieves all members for a project
func (r *GraphProjectRepository) GetProjectMembers(ctx context.Context, projectID string) ([]domain.ProjectMember, error) {
	// Query all ProjectMember nodes and filter by project ID prefix
	filters := map[string]interface{}{}

	memberProps, err := r.graph.QueryNodes(ctx, NodeTypeProjectMember, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to query project members: %w", err)
	}

	var members []domain.ProjectMember
	for _, props := range memberProps {
		// Check if this member belongs to our project (simple ID check)
		if nodeID, ok := props["id"].(string); ok {
			if len(nodeID) > len(projectID) && nodeID[:len(projectID)] == projectID {
				member, err := r.mapToProjectMember(props)
				if err != nil {
					return nil, fmt.Errorf("failed to map member properties: %w", err)
				}
				members = append(members, *member)
			}
		}
	}

	return members, nil
}

// FindProjectsByOwner finds projects by owner email
func (r *GraphProjectRepository) FindProjectsByOwner(ctx context.Context, ownerEmail string) ([]*domain.Project, error) {
	filters := map[string]interface{}{
		"owner_email": ownerEmail,
	}

	projectProps, err := r.graph.QueryNodes(ctx, NodeTypeProject, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to query projects by owner: %w", err)
	}

	projects := make([]*domain.Project, len(projectProps))
	for i, props := range projectProps {
		project, err := r.mapToProject(props)
		if err != nil {
			return nil, fmt.Errorf("failed to map project properties: %w", err)
		}
		projects[i] = project
	}

	return projects, nil
}

// FindProjectsByMember finds projects where a user is a member
func (r *GraphProjectRepository) FindProjectsByMember(ctx context.Context, userID string) ([]*domain.Project, error) {
	// Query members first to find project IDs
	filters := map[string]interface{}{
		"user_id": userID,
	}

	memberProps, err := r.graph.QueryNodes(ctx, NodeTypeProjectMember, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to query members: %w", err)
	}

	// Extract project IDs from member node IDs
	projectIDs := make(map[string]bool)
	for _, props := range memberProps {
		if nodeID, ok := props["id"].(string); ok {
			// Extract project ID from composite ID (projectID_userID)
			if underscoreIndex := len(nodeID) - len(userID) - 1; underscoreIndex > 0 {
				projectID := nodeID[:underscoreIndex]
				projectIDs[projectID] = true
			}
		}
	}

	// Get projects for found project IDs
	var projects []*domain.Project
	for projectID := range projectIDs {
		project, err := r.GetProject(ctx, projectID)
		if err != nil {
			continue // Skip if project not found
		}
		projects = append(projects, project)
	}

	return projects, nil
}

// FindProjectsByStatus finds projects by status
func (r *GraphProjectRepository) FindProjectsByStatus(ctx context.Context, status domain.ProjectStatus) ([]*domain.Project, error) {
	filters := map[string]interface{}{
		"status": string(status),
	}

	projectProps, err := r.graph.QueryNodes(ctx, NodeTypeProject, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to query projects by status: %w", err)
	}

	projects := make([]*domain.Project, len(projectProps))
	for i, props := range projectProps {
		project, err := r.mapToProject(props)
		if err != nil {
			return nil, fmt.Errorf("failed to map project properties: %w", err)
		}
		projects[i] = project
	}

	return projects, nil
}

// GetAllProjects retrieves all projects
func (r *GraphProjectRepository) GetAllProjects(ctx context.Context) ([]*domain.Project, error) {
	filters := map[string]interface{}{}

	projectProps, err := r.graph.QueryNodes(ctx, NodeTypeProject, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to query all projects: %w", err)
	}

	projects := make([]*domain.Project, len(projectProps))
	for i, props := range projectProps {
		project, err := r.mapToProject(props)
		if err != nil {
			return nil, fmt.Errorf("failed to map project properties: %w", err)
		}
		projects[i] = project
	}

	return projects, nil
}

// mapToProject converts map properties to Project domain object
func (r *GraphProjectRepository) mapToProject(props map[string]interface{}) (*domain.Project, error) {
	id, ok := props["id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid project id")
	}

	name, ok := props["name"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid project name")
	}

	ownerEmail, ok := props["owner_email"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid project owner_email")
	}

	description, _ := props["description"].(string)
	statusStr, _ := props["status"].(string)

	createdAtStr, ok := props["created_at"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid created_at")
	}

	updatedAtStr, ok := props["updated_at"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid updated_at")
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

	project := &domain.Project{
		ID:          id,
		Name:        name,
		Description: description,
		OwnerEmail:  ownerEmail,
		Status:      domain.ProjectStatus(statusStr),
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
		Members:     []domain.ProjectMember{},
	}

	return project, nil
}

// mapToProjectMember converts map properties to ProjectMember domain object
func (r *GraphProjectRepository) mapToProjectMember(props map[string]interface{}) (*domain.ProjectMember, error) {
	userID, ok := props["user_id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid member user_id")
	}

	email, ok := props["email"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid member email")
	}

	roleStr, ok := props["role"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid member role")
	}

	member := &domain.ProjectMember{
		UserID: userID,
		Email:  email,
		Role:   domain.ProjectRole(roleStr),
	}

	return member, nil
}
