package application

import (
	"context"
	"errors"
	"testing"

	"neuromesh/internal/project/domain"
)

// MockProjectRepository for testing
type MockProjectRepository struct {
	projects map[string]*domain.Project
	members  map[string][]domain.ProjectMember
}

func NewMockProjectRepository() *MockProjectRepository {
	return &MockProjectRepository{
		projects: make(map[string]*domain.Project),
		members:  make(map[string][]domain.ProjectMember),
	}
}

func (m *MockProjectRepository) CreateProject(ctx context.Context, project *domain.Project) error {
	m.projects[project.ID] = project
	return nil
}

func (m *MockProjectRepository) GetProject(ctx context.Context, projectID string) (*domain.Project, error) {
	project, exists := m.projects[projectID]
	if !exists {
		return nil, errors.New("project not found")
	}
	return project, nil
}

func (m *MockProjectRepository) GetProjectWithMembers(ctx context.Context, projectID string) (*domain.Project, error) {
	project, err := m.GetProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	// For simplicity, return project with members already populated
	if members, exists := m.members[projectID]; exists {
		project.Members = members
	}
	return project, nil
}

func (m *MockProjectRepository) UpdateProject(ctx context.Context, project *domain.Project) error {
	m.projects[project.ID] = project
	return nil
}

func (m *MockProjectRepository) DeleteProject(ctx context.Context, projectID string) error {
	delete(m.projects, projectID)
	return nil
}

func (m *MockProjectRepository) GetProjectMembers(ctx context.Context, projectID string) ([]domain.ProjectMember, error) {
	members, exists := m.members[projectID]
	if !exists {
		return []domain.ProjectMember{}, nil
	}
	return members, nil
}

func (m *MockProjectRepository) FindProjectsByOwner(ctx context.Context, ownerEmail string) ([]*domain.Project, error) {
	var projects []*domain.Project
	for _, project := range m.projects {
		if project.OwnerEmail == ownerEmail {
			projects = append(projects, project)
		}
	}
	return projects, nil
}

func (m *MockProjectRepository) FindProjectsByMember(ctx context.Context, userID string) ([]*domain.Project, error) {
	var projects []*domain.Project
	for _, project := range m.projects {
		for _, member := range project.Members {
			if member.UserID == userID {
				projects = append(projects, project)
				break
			}
		}
	}
	return projects, nil
}

func (m *MockProjectRepository) FindProjectsByStatus(ctx context.Context, status domain.ProjectStatus) ([]*domain.Project, error) {
	var projects []*domain.Project
	for _, project := range m.projects {
		if project.Status == status {
			projects = append(projects, project)
		}
	}
	return projects, nil
}

func (m *MockProjectRepository) AddMember(ctx context.Context, projectID string, member *domain.ProjectMember) error {
	if members, exists := m.members[projectID]; exists {
		m.members[projectID] = append(members, *member)
	} else {
		m.members[projectID] = []domain.ProjectMember{*member}
	}
	return nil
}

func (m *MockProjectRepository) RemoveMember(ctx context.Context, projectID, userID string) error {
	if members, exists := m.members[projectID]; exists {
		for i, member := range members {
			if member.UserID == userID {
				m.members[projectID] = append(members[:i], members[i+1:]...)
				break
			}
		}
	}
	return nil
}

func (m *MockProjectRepository) GetAllProjects(ctx context.Context) ([]*domain.Project, error) {
	var projects []*domain.Project
	for _, project := range m.projects {
		projects = append(projects, project)
	}
	return projects, nil
}

func (m *MockProjectRepository) EnsureProjectSchema(ctx context.Context) error {
	return nil
}

func TestCreateProject(t *testing.T) {
	repo := NewMockProjectRepository()
	service := NewProjectService(repo)
	ctx := context.Background()

	tests := []struct {
		name        string
		id          string
		projectName string
		ownerEmail  string
		wantErr     bool
	}{
		{
			name:        "valid project creation",
			id:          "project-123",
			projectName: "Test Project",
			ownerEmail:  "owner@example.com",
			wantErr:     false,
		},
		{
			name:        "empty project name",
			id:          "project-124",
			projectName: "",
			ownerEmail:  "owner@example.com",
			wantErr:     true,
		},
		{
			name:        "empty owner email",
			id:          "project-125",
			projectName: "Test Project",
			ownerEmail:  "",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			project, err := service.CreateProject(ctx, tt.id, tt.projectName, tt.ownerEmail)

			if tt.wantErr {
				if err == nil {
					t.Errorf("CreateProject() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("CreateProject() unexpected error: %v", err)
				return
			}

			if project.ID != tt.id {
				t.Errorf("CreateProject() project ID = %v, want %v", project.ID, tt.id)
			}

			if project.Name != tt.projectName {
				t.Errorf("CreateProject() project name = %v, want %v", project.Name, tt.projectName)
			}

			if project.OwnerEmail != tt.ownerEmail {
				t.Errorf("CreateProject() owner email = %v, want %v", project.OwnerEmail, tt.ownerEmail)
			}

			if project.Status != domain.ProjectStatusActive {
				t.Errorf("CreateProject() project status = %v, want %v", project.Status, domain.ProjectStatusActive)
			}
		})
	}
}

func TestGetProject(t *testing.T) {
	repo := NewMockProjectRepository()
	service := NewProjectService(repo)
	ctx := context.Background()

	// Create a test project
	testProject, err := domain.NewProject("project-123", "Test Project", "owner@example.com")
	if err != nil {
		t.Fatalf("Failed to create test project: %v", err)
	}
	repo.projects["project-123"] = testProject

	tests := []struct {
		name      string
		projectID string
		wantErr   bool
	}{
		{
			name:      "existing project",
			projectID: "project-123",
			wantErr:   false,
		},
		{
			name:      "non-existing project",
			projectID: "project-404",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			project, err := service.GetProject(ctx, tt.projectID)

			if tt.wantErr {
				if err == nil {
					t.Errorf("GetProject() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("GetProject() unexpected error: %v", err)
				return
			}

			if project.ID != tt.projectID {
				t.Errorf("GetProject() project ID = %v, want %v", project.ID, tt.projectID)
			}
		})
	}
}

func TestAddMember(t *testing.T) {
	repo := NewMockProjectRepository()
	service := NewProjectService(repo)
	ctx := context.Background()

	// Create a test project
	testProject, err := domain.NewProject("project-123", "Test Project", "owner@example.com")
	if err != nil {
		t.Fatalf("Failed to create test project: %v", err)
	}
	repo.projects["project-123"] = testProject

	tests := []struct {
		name      string
		projectID string
		userID    string
		email     string
		role      domain.ProjectRole
		wantErr   bool
	}{
		{
			name:      "valid member addition",
			projectID: "project-123",
			userID:    "user-456",
			email:     "member@example.com",
			role:      domain.ProjectRoleMember,
			wantErr:   false,
		},
		{
			name:      "duplicate member",
			projectID: "project-123",
			userID:    "user-456",
			email:     "member@example.com",
			role:      domain.ProjectRoleMember,
			wantErr:   true,
		},
		{
			name:      "non-existing project",
			projectID: "project-404",
			userID:    "user-789",
			email:     "member2@example.com",
			role:      domain.ProjectRoleMember,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.AddMember(ctx, tt.projectID, tt.userID, tt.email, tt.role)

			if tt.wantErr {
				if err == nil {
					t.Errorf("AddMember() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("AddMember() unexpected error: %v", err)
				return
			}

			// Verify member was added
			project, err := service.GetProject(ctx, tt.projectID)
			if err != nil {
				t.Errorf("Failed to get project after adding member: %v", err)
				return
			}

			found := false
			for _, member := range project.Members {
				if member.UserID == tt.userID {
					found = true
					if member.Email != tt.email {
						t.Errorf("Member email = %v, want %v", member.Email, tt.email)
					}
					if member.Role != tt.role {
						t.Errorf("Member role = %v, want %v", member.Role, tt.role)
					}
					break
				}
			}

			if !found {
				t.Errorf("Member %v not found in project", tt.userID)
			}
		})
	}
}

func TestUpdateProjectDescription(t *testing.T) {
	repo := NewMockProjectRepository()
	service := NewProjectService(repo)
	ctx := context.Background()

	// Create a test project
	testProject, err := domain.NewProject("project-123", "Test Project", "owner@example.com")
	if err != nil {
		t.Fatalf("Failed to create test project: %v", err)
	}
	repo.projects["project-123"] = testProject

	tests := []struct {
		name        string
		projectID   string
		description string
		wantErr     bool
	}{
		{
			name:        "valid description update",
			projectID:   "project-123",
			description: "Updated description",
			wantErr:     false,
		},
		{
			name:        "non-existing project",
			projectID:   "project-404",
			description: "Some description",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.UpdateProjectDescription(ctx, tt.projectID, tt.description)

			if tt.wantErr {
				if err == nil {
					t.Errorf("UpdateProjectDescription() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("UpdateProjectDescription() unexpected error: %v", err)
				return
			}

			// Verify description was updated
			project, err := service.GetProject(ctx, tt.projectID)
			if err != nil {
				t.Errorf("Failed to get project after updating description: %v", err)
				return
			}

			if project.Description != tt.description {
				t.Errorf("Project description = %v, want %v", project.Description, tt.description)
			}
		})
	}
}

func TestFindProjectsByOwner(t *testing.T) {
	repo := NewMockProjectRepository()
	service := NewProjectService(repo)
	ctx := context.Background()

	// Create test projects
	project1, _ := domain.NewProject("project-1", "Project 1", "owner1@example.com")
	project2, _ := domain.NewProject("project-2", "Project 2", "owner2@example.com")
	project3, _ := domain.NewProject("project-3", "Project 3", "owner1@example.com")

	repo.projects["project-1"] = project1
	repo.projects["project-2"] = project2
	repo.projects["project-3"] = project3

	projects, err := service.FindProjectsByOwner(ctx, "owner1@example.com")
	if err != nil {
		t.Errorf("FindProjectsByOwner() unexpected error: %v", err)
		return
	}

	if len(projects) != 2 {
		t.Errorf("FindProjectsByOwner() found %d projects, want 2", len(projects))
		return
	}

	// Verify the correct projects were returned
	projectIDs := make(map[string]bool)
	for _, project := range projects {
		projectIDs[project.ID] = true
	}

	if !projectIDs["project-1"] || !projectIDs["project-3"] {
		t.Errorf("FindProjectsByOwner() returned incorrect projects")
	}
}
