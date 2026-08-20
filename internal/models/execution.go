package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ExecutionStatus string

const (
	ExecutionStatusPending ExecutionStatus = "pending"
	ExecutionStatusRunning ExecutionStatus = "running"
	ExecutionStatusSuccess ExecutionStatus = "success"
	ExecutionStatusFailed  ExecutionStatus = "failed"
)

type Execution struct {
	ID        uuid.UUID       `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	ProjectID uuid.UUID       `gorm:"type:uuid;not null;index" json:"project_id"`
	CommitSHA string          `gorm:"not null" json:"commit_sha"`
	Status    ExecutionStatus `gorm:"type:varchar(20);not null;default:'pending'" json:"status"`
	Logs      string          `gorm:"type:text" json:"logs"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
	DeletedAt gorm.DeletedAt  `gorm:"index" json:"-"`

	Project *Project `gorm:"foreignKey:ProjectID" json:"-"`
}
