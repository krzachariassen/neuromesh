package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewProject tests project creation with validation - RED phase
func TestNewProject(t *testing.T) {
	t.Run("should_create_valid_project", func(t *testing.T) {
		// Act: Create a new project
		project, err := NewProject("proj-123", "Test Project", "test@example.com")

		// Assert: Should create valid project
		require.NoError(t, err)
		assert.Equal(t, "proj-123", project.ID)
		assert.Equal(t, "Test Project", project.Name)
		assert.Equal(t, "test@example.com", project.OwnerEmail)
		assert.Equal(t, ProjectStatusActive, project.Status)
		assert.False(t, project.CreatedAt.IsZero())
		assert.False(t, project.UpdatedAt.IsZero())
		assert.NotNil(t, project.Metadata)
	})

	t.Run("should_fail_with_empty_project_ID", func(t *testing.T) {
		// Act: Try to create project with empty ID
		project, err := NewProject("", "Test Project", "test@example.com")

		// Assert: Should fail validation
		require.Error(t, err)
		assert.Nil(t, project)
		assert.Contains(t, err.Error(), "project id cannot be empty")
	})

	t.Run("should_fail_with_empty_name", func(t *testing.T) {
		// Act: Try to create project with empty name
		project, err := NewProject("proj-123", "", "test@example.com")

		// Assert: Should fail validation
		require.Error(t, err)
		assert.Nil(t, project)
		assert.Contains(t, err.Error(), "project name cannot be empty")
	})

	t.Run("should_fail_with_empty_owner_email", func(t *testing.T) {
		// Act: Try to create project with empty owner email
		project, err := NewProject("proj-123", "Test Project", "")

		// Assert: Should fail validation
		require.Error(t, err)
		assert.Nil(t, project)
		assert.Contains(t, err.Error(), "owner email cannot be empty")
	})
}

// TestProject_UpdateDescription tests project description updates
func TestProject_UpdateDescription(t *testing.T) {
	t.Run("should_update_description", func(t *testing.T) {
		// Setup: Create project
		project, err := NewProject("proj-123", "Test Project", "test@example.com")
		require.NoError(t, err)

		// Act: Update description
		err = project.UpdateDescription("Updated description")

		// Assert: Should update description and timestamp
		require.NoError(t, err)
		assert.Equal(t, "Updated description", project.Description)
		assert.True(t, project.UpdatedAt.After(project.CreatedAt))
	})
}

// TestProject_AddMember tests project member management
func TestProject_AddMember(t *testing.T) {
	t.Run("should_add_member", func(t *testing.T) {
		// Setup: Create project
		project, err := NewProject("proj-123", "Test Project", "owner@example.com")
		require.NoError(t, err)

		// Act: Add member
		err = project.AddMember("user-456", "member@example.com", ProjectRoleMember)

		// Assert: Should add member
		require.NoError(t, err)
		assert.Len(t, project.Members, 1)
		assert.Equal(t, "user-456", project.Members[0].UserID)
		assert.Equal(t, "member@example.com", project.Members[0].Email)
		assert.Equal(t, ProjectRoleMember, project.Members[0].Role)
	})

	t.Run("should_fail_to_add_duplicate_member", func(t *testing.T) {
		// Setup: Create project with member
		project, err := NewProject("proj-123", "Test Project", "owner@example.com")
		require.NoError(t, err)
		err = project.AddMember("user-456", "member@example.com", ProjectRoleMember)
		require.NoError(t, err)

		// Act: Try to add same member again
		err = project.AddMember("user-456", "member@example.com", ProjectRoleMember)

		// Assert: Should fail
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user already member of project")
	})
}
