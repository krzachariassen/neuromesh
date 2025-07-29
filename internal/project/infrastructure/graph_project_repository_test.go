package infrastructure

import (
	"context"
	"testing"
	"time"

	"neuromesh/internal/project/domain"
	"neuromesh/testHelpers"
)

func TestGraphProjectRepository_Integration(t *testing.T) {
	// Skip if not in integration test mode
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	graph := testHelpers.NewCleanMockGraph()

	repo := NewGraphProjectRepository(graph)

	// Ensure schema
	err := repo.EnsureProjectSchema(ctx)
	if err != nil {
		t.Fatalf("Failed to ensure schema: %v", err)
	}

	// Test project creation
	project, err := domain.NewProject("test-project-1", "Test Project", "owner@example.com")
	if err != nil {
		t.Fatalf("Failed to create domain project: %v", err)
	}

	err = repo.CreateProject(ctx, project)
	if err != nil {
		t.Fatalf("Failed to create project in graph: %v", err)
	}

	// Test project retrieval
	retrievedProject, err := repo.GetProject(ctx, "test-project-1")
	if err != nil {
		t.Fatalf("Failed to get project: %v", err)
	}

	if retrievedProject.ID != project.ID {
		t.Errorf("Project ID mismatch: got %s, want %s", retrievedProject.ID, project.ID)
	}

	if retrievedProject.Name != project.Name {
		t.Errorf("Project name mismatch: got %s, want %s", retrievedProject.Name, project.Name)
	}

	if retrievedProject.OwnerEmail != project.OwnerEmail {
		t.Errorf("Project owner email mismatch: got %s, want %s", retrievedProject.OwnerEmail, project.OwnerEmail)
	}

	// Test member addition
	member := &domain.ProjectMember{
		UserID: "user-123",
		Email:  "member@example.com",
		Role:   domain.ProjectRoleMember,
	}

	err = repo.AddMember(ctx, "test-project-1", member)
	if err != nil {
		t.Fatalf("Failed to add member: %v", err)
	}

	// Test get project with members
	projectWithMembers, err := repo.GetProjectWithMembers(ctx, "test-project-1")
	if err != nil {
		t.Fatalf("Failed to get project with members: %v", err)
	}

	if len(projectWithMembers.Members) != 1 {
		t.Errorf("Expected 1 member, got %d", len(projectWithMembers.Members))
	}

	if len(projectWithMembers.Members) > 0 {
		retrievedMember := projectWithMembers.Members[0]
		if retrievedMember.UserID != member.UserID {
			t.Errorf("Member UserID mismatch: got %s, want %s", retrievedMember.UserID, member.UserID)
		}
		if retrievedMember.Email != member.Email {
			t.Errorf("Member email mismatch: got %s, want %s", retrievedMember.Email, member.Email)
		}
		if retrievedMember.Role != member.Role {
			t.Errorf("Member role mismatch: got %s, want %s", retrievedMember.Role, member.Role)
		}
	}

	// Test project update
	project.UpdateDescription("Updated description")
	project.UpdatedAt = time.Now().UTC()

	err = repo.UpdateProject(ctx, project)
	if err != nil {
		t.Fatalf("Failed to update project: %v", err)
	}

	updatedProject, err := repo.GetProject(ctx, "test-project-1")
	if err != nil {
		t.Fatalf("Failed to get updated project: %v", err)
	}

	if updatedProject.Description != "Updated description" {
		t.Errorf("Project description not updated: got %s, want %s", updatedProject.Description, "Updated description")
	}

	// Test find projects by owner
	projects, err := repo.FindProjectsByOwner(ctx, "owner@example.com")
	if err != nil {
		t.Fatalf("Failed to find projects by owner: %v", err)
	}

	if len(projects) != 1 {
		t.Errorf("Expected 1 project for owner, got %d", len(projects))
	}

	// Test find projects by member
	memberProjects, err := repo.FindProjectsByMember(ctx, "user-123")
	if err != nil {
		t.Fatalf("Failed to find projects by member: %v", err)
	}

	if len(memberProjects) != 1 {
		t.Errorf("Expected 1 project for member, got %d", len(memberProjects))
	}

	// Test member removal
	err = repo.RemoveMember(ctx, "test-project-1", "user-123")
	if err != nil {
		t.Fatalf("Failed to remove member: %v", err)
	}

	members, err := repo.GetProjectMembers(ctx, "test-project-1")
	if err != nil {
		t.Fatalf("Failed to get project members: %v", err)
	}

	if len(members) != 0 {
		t.Errorf("Expected 0 members after removal, got %d", len(members))
	}

	// Test project deletion
	err = repo.DeleteProject(ctx, "test-project-1")
	if err != nil {
		t.Fatalf("Failed to delete project: %v", err)
	}

	// Verify project is deleted
	_, err = repo.GetProject(ctx, "test-project-1")
	if err == nil {
		t.Error("Expected error when getting deleted project, but got none")
	}
}
