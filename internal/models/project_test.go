package models

import (
	"testing"
)

func TestNewProject(t *testing.T) {
	name := "Test Project"
	project := NewProject(name)

	if project.Name != name {
		t.Errorf("expected project name to be %s, got %s", name, project.Name)
	}

	if project.ID.String() == "" {
		t.Error("expected project ID to be generated")
	}

	if project.CreatedAt.IsZero() || project.UpdatedAt.IsZero() {
		t.Error("expected timestamps to be set")
	}
}

func TestNewLLMCredential(t *testing.T) {
	project := NewProject("Tenant A")
	cred := NewLLMCredential(project.ID, ProviderOpenAI, "sk-test123")

	if cred.ProjectID != project.ID {
		t.Errorf("expected credential to be linked to project ID %s", project.ID)
	}

	if cred.Provider != ProviderOpenAI {
		t.Errorf("expected provider to be %s, got %s", ProviderOpenAI, cred.Provider)
	}
}
