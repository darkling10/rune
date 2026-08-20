package ai

import (
	"testing"

	"github.com/deployerai/deployer/internal/models"
	"github.com/google/uuid"
)

func TestFactory(t *testing.T) {
	tests := []struct {
		name        string
		provider    models.ProviderType
		expectError bool
	}{
		{"OpenAI Provider", models.ProviderOpenAI, false},
		{"Claude Provider", models.ProviderClaude, false},
		{"Bedrock Provider", models.ProviderBedrock, false},
		{"Unknown Provider", models.ProviderType("unknown"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cred := &models.LLMCredential{
				ID:        uuid.New(),
				ProjectID: uuid.New(),
				Provider:  tt.provider,
				APIKey:    "test-key",
			}

			provider, err := Factory(cred)

			if tt.expectError {
				if err == nil {
					t.Error("expected an error for unknown provider, got none")
				}
				if provider != nil {
					t.Error("expected provider to be nil on error")
				}
			} else {
				if err != nil {
					t.Errorf("did not expect error, got %v", err)
				}
				if provider == nil {
					t.Error("expected provider to be created")
				}
			}
		})
	}
}
