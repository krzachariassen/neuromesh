package application

import (
	"context"
	"fmt"

	"neuromesh/internal/project/domain"
)

// ProjectService defines the application service interface for project management
type ProjectService interface {
	// Project management
	CreateProject(ctx context.Context, id, name, ownerEmail string) (*domain.Project, error)
	GetProject(ctx context.Context, projectID string) (*domain.Project, error)
	GetProjectWithMembers(ctx context.Context, projectID string) (*domain.Project, error)
	UpdateProjectDescription(ctx context.Context, projectID, description string) error
	UpdateProjectStatus(ctx context.Context, projectID string, status domain.ProjectStatus) error
	DeleteProject(ctx context.Context, projectID string) error

	// Member management
	AddMember(ctx context.Context, projectID, userID, email string, role domain.ProjectRole) error
	RemoveMember(ctx context.Context, projectID, userID string) error
	GetProjectMembers(ctx context.Context, projectID string) ([]domain.ProjectMember, error)

	// Query operations
	FindProjectsByOwner(ctx context.Context, ownerEmail string) ([]*domain.Project, error)
	FindProjectsByMember(ctx context.Context, userID string) ([]*domain.Project, error)
	FindActiveProjects(ctx context.Context) ([]*domain.Project, error)

	// Schema management
	EnsureSchema(ctx context.Context) error
}

// ProjectServiceImpl implements the ProjectService interface
type ProjectServiceImpl struct {
	repo domain.ProjectRepository
}

// NewProjectService creates a new project service implementation
func NewProjectService(repo domain.ProjectRepository) ProjectService {
	return &ProjectServiceImpl{
		repo: repo,
	}
}

// CreateProject creates a new project
func (s *ProjectServiceImpl) CreateProject(ctx context.Context, id, name, ownerEmail string) (*domain.Project, error) {
	project, err := domain.NewProject(id, name, ownerEmail)
	if err != nil {
		return nil, fmt.Errorf("failed to create project domain object: %w", err)
	}

	if err := s.repo.CreateProject(ctx, project); err != nil {
		return nil, fmt.Errorf("failed to store project: %w", err)
	}

	return project, nil
}

// GetProject retrieves a project by ID
func (s *ProjectServiceImpl) GetProject(ctx context.Context, projectID string) (*domain.Project, error) {
	project, err := s.repo.GetProject(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}
	return project, nil
}

// GetProjectWithMembers retrieves a project with all its members
func (s *ProjectServiceImpl) GetProjectWithMembers(ctx context.Context, projectID string) (*domain.Project, error) {
	project, err := s.repo.GetProjectWithMembers(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get project with members: %w", err)
	}
	return project, nil
}

// UpdateProjectDescription updates a project's description
func (s *ProjectServiceImpl) UpdateProjectDescription(ctx context.Context, projectID, description string) error {
	project, err := s.repo.GetProject(ctx, projectID)
	if err != nil {
		return fmt.Errorf("failed to get project: %w", err)
	}

	if err := project.UpdateDescription(description); err != nil {
		return fmt.Errorf("failed to update description: %w", err)
	}

	if err := s.repo.UpdateProject(ctx, project); err != nil {
		return fmt.Errorf("failed to save project: %w", err)
	}

	return nil
}

// UpdateProjectStatus updates a project's status
func (s *ProjectServiceImpl) UpdateProjectStatus(ctx context.Context, projectID string, status domain.ProjectStatus) error {
	project, err := s.repo.GetProject(ctx, projectID)
	if err != nil {
		return fmt.Errorf("failed to get project: %w", err)
	}

	project.SetStatus(status)

	if err := s.repo.UpdateProject(ctx, project); err != nil {
		return fmt.Errorf("failed to save project: %w", err)
	}

	return nil
}

// DeleteProject deletes a project
func (s *ProjectServiceImpl) DeleteProject(ctx context.Context, projectID string) error {
	if err := s.repo.DeleteProject(ctx, projectID); err != nil {
		return fmt.Errorf("failed to delete project: %w", err)
	}
	return nil
}

// AddMember adds a member to a project
func (s *ProjectServiceImpl) AddMember(ctx context.Context, projectID, userID, email string, role domain.ProjectRole) error {
	project, err := s.repo.GetProject(ctx, projectID)
	if err != nil {
		return fmt.Errorf("failed to get project: %w", err)
	}

	member := &domain.ProjectMember{
		UserID: userID,
		Email:  email,
		Role:   role,
	}

	if err := project.AddMember(userID, email, role); err != nil {
		return fmt.Errorf("failed to add member to project: %w", err)
	}

	if err := s.repo.AddMember(ctx, projectID, member); err != nil {
		return fmt.Errorf("failed to persist member: %w", err)
	}

	return nil
}

// RemoveMember removes a member from a project
func (s *ProjectServiceImpl) RemoveMember(ctx context.Context, projectID, userID string) error {
	project, err := s.repo.GetProject(ctx, projectID)
	if err != nil {
		return fmt.Errorf("failed to get project: %w", err)
	}

	if err := project.RemoveMember(userID); err != nil {
		return fmt.Errorf("failed to remove member from project: %w", err)
	}

	if err := s.repo.UpdateProject(ctx, project); err != nil {
		return fmt.Errorf("failed to save project: %w", err)
	}

	return nil
}

// GetProjectMembers retrieves all members for a project
func (s *ProjectServiceImpl) GetProjectMembers(ctx context.Context, projectID string) ([]domain.ProjectMember, error) {
	members, err := s.repo.GetProjectMembers(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get project members: %w", err)
	}
	return members, nil
}

// FindProjectsByOwner finds projects by owner email
func (s *ProjectServiceImpl) FindProjectsByOwner(ctx context.Context, ownerEmail string) ([]*domain.Project, error) {
	projects, err := s.repo.FindProjectsByOwner(ctx, ownerEmail)
	if err != nil {
		return nil, fmt.Errorf("failed to find projects by owner: %w", err)
	}
	return projects, nil
}

// FindProjectsByMember finds projects by member user ID
func (s *ProjectServiceImpl) FindProjectsByMember(ctx context.Context, userID string) ([]*domain.Project, error) {
	projects, err := s.repo.FindProjectsByMember(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to find projects by member: %w", err)
	}
	return projects, nil
}

// FindActiveProjects finds all active projects
func (s *ProjectServiceImpl) FindActiveProjects(ctx context.Context) ([]*domain.Project, error) {
	projects, err := s.repo.FindProjectsByStatus(ctx, domain.ProjectStatusActive)
	if err != nil {
		return nil, fmt.Errorf("failed to find active projects: %w", err)
	}
	return projects, nil
}

// EnsureSchema ensures the project schema is in place
func (s *ProjectServiceImpl) EnsureSchema(ctx context.Context) error {
	if err := s.repo.EnsureProjectSchema(ctx); err != nil {
		return fmt.Errorf("failed to ensure project schema: %w", err)
	}
	return nil
}
