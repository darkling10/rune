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

	CreateExecution(ctx context.Context, exec *models.Execution) error
	UpdateExecutionStatus(ctx context.Context, id uuid.UUID, status models.ExecutionStatus) error
	AppendExecutionLog(ctx context.Context, id uuid.UUID, log string) error
	ListExecutions(ctx context.Context, projectID uuid.UUID) ([]models.Execution, error)
	GetExecution(ctx context.Context, id uuid.UUID) (*models.Execution, error)
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

func (r *projectRepo) CreateExecution(ctx context.Context, exec *models.Execution) error {
	return r.db.WithContext(ctx).Create(exec).Error
}

func (r *projectRepo) UpdateExecutionStatus(ctx context.Context, id uuid.UUID, status models.ExecutionStatus) error {
	return r.db.WithContext(ctx).Model(&models.Execution{}).Where("id = ?", id).Update("status", status).Error
}

func (r *projectRepo) AppendExecutionLog(ctx context.Context, id uuid.UUID, log string) error {
	// Simple append in Postgres. For scale, we'd use unnest or a separate table, but this is fine for MVP.
	return r.db.WithContext(ctx).Model(&models.Execution{}).Where("id = ?", id).
		Update("logs", gorm.Expr("logs || ?", log+"\n")).Error
}

func (r *projectRepo) ListExecutions(ctx context.Context, projectID uuid.UUID) ([]models.Execution, error) {
	var execs []models.Execution
	err := r.db.WithContext(ctx).Select("id, project_id, commit_sha, status, created_at, updated_at").
		Where("project_id = ?", projectID).Order("created_at desc").Find(&execs).Error
	return execs, err
}

func (r *projectRepo) GetExecution(ctx context.Context, id uuid.UUID) (*models.Execution, error) {
	var exec models.Execution
	err := r.db.WithContext(ctx).First(&exec, "id = ?", id).Error
	return &exec, err
}
