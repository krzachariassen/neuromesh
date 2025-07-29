package domain

import (
	"fmt"
	"time"
)

// ProjectValidationError represents validation errors for projects
type ProjectValidationError struct {
	Field   string
	Message string
}

func (e ProjectValidationError) Error() string {
	return fmt.Sprintf("project validation error - %s: %s", e.Field, e.Message)
}

// ProjectStatus represents the status of a project
type ProjectStatus string

const (
	ProjectStatusActive   ProjectStatus = "active"
	ProjectStatusInactive ProjectStatus = "inactive"
	ProjectStatusArchived ProjectStatus = "archived"
)

// ProjectRole represents the role of a member in a project
type ProjectRole string

const (
	ProjectRoleOwner  ProjectRole = "owner"
	ProjectRoleAdmin  ProjectRole = "admin"
	ProjectRoleMember ProjectRole = "member"
	ProjectRoleViewer ProjectRole = "viewer"
)

// ProjectMember represents a member of a project
type ProjectMember struct {
	UserID  string      `json:"user_id"`
	Email   string      `json:"email"`
	Role    ProjectRole `json:"role"`
	AddedAt time.Time   `json:"added_at"`
	AddedBy string      `json:"added_by"`
}

// Project represents a project in the SaaS platform
type Project struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	OwnerEmail  string                 `json:"owner_email"`
	Status      ProjectStatus          `json:"status"`
	Members     []ProjectMember        `json:"members"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// NewProject creates a new project with validation
func NewProject(id, name, ownerEmail string) (*Project, error) {
	// Validation
	if id == "" {
		return nil, ProjectValidationError{
			Field:   "id",
			Message: "project id cannot be empty",
		}
	}

	if name == "" {
		return nil, ProjectValidationError{
			Field:   "name",
			Message: "project name cannot be empty",
		}
	}

	if ownerEmail == "" {
		return nil, ProjectValidationError{
			Field:   "owner_email",
			Message: "owner email cannot be empty",
		}
	}

	now := time.Now().UTC()

	return &Project{
		ID:         id,
		Name:       name,
		OwnerEmail: ownerEmail,
		Status:     ProjectStatusActive,
		Members:    make([]ProjectMember, 0),
		CreatedAt:  now,
		UpdatedAt:  now,
		Metadata:   make(map[string]interface{}),
	}, nil
}

// UpdateDescription updates the project description
func (p *Project) UpdateDescription(description string) error {
	p.Description = description
	p.UpdatedAt = time.Now().UTC()
	return nil
}

// AddMember adds a member to the project
func (p *Project) AddMember(userID, email string, role ProjectRole) error {
	// Check if user is already a member
	for _, member := range p.Members {
		if member.UserID == userID {
			return ProjectValidationError{
				Field:   "user_id",
				Message: "user already member of project",
			}
		}
	}

	// Add the member
	member := ProjectMember{
		UserID:  userID,
		Email:   email,
		Role:    role,
		AddedAt: time.Now().UTC(),
		AddedBy: p.OwnerEmail, // For now, assume owner adds members
	}

	p.Members = append(p.Members, member)
	p.UpdatedAt = time.Now().UTC()

	return nil
}

// GetMember gets a project member by user ID
func (p *Project) GetMember(userID string) (*ProjectMember, error) {
	for _, member := range p.Members {
		if member.UserID == userID {
			return &member, nil
		}
	}
	return nil, fmt.Errorf("member not found: %s", userID)
}

// RemoveMember removes a member from the project
func (p *Project) RemoveMember(userID string) error {
	for i, member := range p.Members {
		if member.UserID == userID {
			// Remove member by swapping with last element and truncating
			p.Members[i] = p.Members[len(p.Members)-1]
			p.Members = p.Members[:len(p.Members)-1]
			p.UpdatedAt = time.Now().UTC()
			return nil
		}
	}
	return fmt.Errorf("member not found: %s", userID)
}

// SetStatus updates the project status
func (p *Project) SetStatus(status ProjectStatus) {
	p.Status = status
	p.UpdatedAt = time.Now().UTC()
}

// SetMetadata sets a metadata key-value pair
func (p *Project) SetMetadata(key string, value interface{}) {
	if p.Metadata == nil {
		p.Metadata = make(map[string]interface{})
	}
	p.Metadata[key] = value
	p.UpdatedAt = time.Now().UTC()
}
