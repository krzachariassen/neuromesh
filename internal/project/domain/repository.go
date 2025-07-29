package domain

import "context"

// ProjectRepository defines the interface for project persistence operations
type ProjectRepository interface {
	// Schema management
	EnsureProjectSchema(ctx context.Context) error

	// Project operations
	CreateProject(ctx context.Context, project *Project) error
	GetProject(ctx context.Context, projectID string) (*Project, error)
	GetProjectWithMembers(ctx context.Context, projectID string) (*Project, error)
	UpdateProject(ctx context.Context, project *Project) error
	DeleteProject(ctx context.Context, projectID string) error

	// Member operations
	AddMember(ctx context.Context, projectID string, member *ProjectMember) error
	RemoveMember(ctx context.Context, projectID, userID string) error
	GetProjectMembers(ctx context.Context, projectID string) ([]ProjectMember, error)

	// Query operations
	FindProjectsByOwner(ctx context.Context, ownerEmail string) ([]*Project, error)
	FindProjectsByMember(ctx context.Context, userID string) ([]*Project, error)
	FindProjectsByStatus(ctx context.Context, status ProjectStatus) ([]*Project, error)
	GetAllProjects(ctx context.Context) ([]*Project, error)
}
