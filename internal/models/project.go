package models

import (
	"time"

	"github.com/google/uuid"
)

// Project represents a high-level tenant or isolated workspace in the platform.
// A Project can contain multiple Repositories and Pipelines.
type Project struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	Name      string    `gorm:"type:varchar(255);not null;uniqueIndex" json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// NewProject creates a new Project with a generated UUID and current timestamps.
func NewProject(name string) *Project {
	now := time.Now().UTC()
	return &Project{
		ID:        uuid.New(),
		Name:      name,
		CreatedAt: now,
		UpdatedAt: now,
	}
}
