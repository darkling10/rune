package repository

import (
	"context"

	"github.com/deployerai/deployer/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ProjectRepository interface {
	CreateProject(ctx context.Context, project *models.Project) error
	GetProjectByID(ctx context.Context, id uuid.UUID) (*models.Project, error)
	ListProjects(ctx context.Context) ([]models.Project, error)
	CreateCredential(ctx context.Context, cred *models.LLMCredential) error
	GetCredentialByProjectID(ctx context.Context, id uuid.UUID) (*models.LLMCredential, error)
}

type projectRepo struct {
	db *gorm.DB
}

func NewProjectRepository(db *gorm.DB) ProjectRepository {
	return &projectRepo{db: db}
}

func (r *projectRepo) CreateProject(ctx context.Context, project *models.Project) error {
	return r.db.WithContext(ctx).Create(project).Error
}

func (r *projectRepo) GetProjectByID(ctx context.Context, id uuid.UUID) (*models.Project, error) {
	var project models.Project
	err := r.db.WithContext(ctx).First(&project, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &project, nil
}

func (r *projectRepo) ListProjects(ctx context.Context) ([]models.Project, error) {
	var projects []models.Project
	err := r.db.WithContext(ctx).Find(&projects).Error
	return projects, err
}

func (r *projectRepo) CreateCredential(ctx context.Context, cred *models.LLMCredential) error {
	return r.db.WithContext(ctx).Create(cred).Error
}

func (r *projectRepo) GetCredentialByProjectID(ctx context.Context, projectID uuid.UUID) (*models.LLMCredential, error) {
	var cred models.LLMCredential
	// Fetch the most recently added credential for the project
	err := r.db.WithContext(ctx).Order("created_at desc").First(&cred, "project_id = ?", projectID).Error
	if err != nil {
		return nil, err
	}
	return &cred, nil
}
