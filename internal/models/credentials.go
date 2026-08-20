package models

import (
	"time"

	"github.com/google/uuid"
)

// ProviderType represents the supported LLM providers.
type ProviderType string

const (
	ProviderOpenAI  ProviderType = "openai"
	ProviderClaude  ProviderType = "claude"
	ProviderBedrock ProviderType = "bedrock"
)

// LLMCredential stores the API keys and configuration for a specific LLM provider,
// tied to a specific project for isolation.
// In a production scenario, the APIKey should be encrypted at rest using a KMS.
type LLMCredential struct {
	ID        uuid.UUID    `gorm:"type:uuid;primaryKey" json:"id"`
	ProjectID uuid.UUID    `gorm:"type:uuid;not null;index" json:"project_id"`
	Provider  ProviderType `gorm:"type:varchar(50);not null" json:"provider"`
	APIKey    string       `gorm:"type:text;not null" json:"api_key"`   // TODO: Encrypt before persisting
	BaseURL   string       `gorm:"type:text" json:"base_url,omitempty"` // For custom endpoints or Bedrock region
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`

	// Define the foreign key relationship
	Project Project `gorm:"foreignKey:ProjectID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
}

// NewLLMCredential creates a new credential record for a project.
func NewLLMCredential(projectID uuid.UUID, provider ProviderType, apiKey string) *LLMCredential {
	now := time.Now().UTC()
	return &LLMCredential{
		ID:        uuid.New(),
		ProjectID: projectID,
		Provider:  provider,
		APIKey:    apiKey,
		CreatedAt: now,
		UpdatedAt: now,
	}
}
