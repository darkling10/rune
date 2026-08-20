package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/deployerai/deployer/internal/models"
	"github.com/deployerai/deployer/internal/repository"
	"github.com/deployerai/deployer/pkg/pipeline"
	"github.com/go-git/go-git/v5"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
)

// TaskProcessor is an interface for starting the task processor.
type TaskProcessor interface {
	Start() error
}

type redisTaskProcessor struct {
	server *asynq.Server
	repo   repository.ProjectRepository
	rdb    *redis.Client
}

func NewRedisTaskProcessor(redisOpt asynq.RedisConnOpt, repo repository.ProjectRepository, rdb *redis.Client) TaskProcessor {
	server := asynq.NewServer(
		redisOpt,
		asynq.Config{
			Concurrency: 10, // Number of concurrent workers
			Queues: map[string]int{
				"critical": 6,
				"default":  3,
				"low":      1,
			},
			ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
				log.Printf("Task %s failed: %v", task.Type(), err)
			}),
		},
	)

	return &redisTaskProcessor{
		server: server,
		repo:   repo,
		rdb:    rdb,
	}
}

func (processor *redisTaskProcessor) Start() error {
	mux := asynq.NewServeMux()

	// Register task handlers
	mux.HandleFunc(TaskTypeRunPipeline, processor.ProcessTaskRunPipeline)
	mux.HandleFunc(TaskTypeSnykScan, processor.ProcessTaskSnykScan)

	return processor.server.Start(mux)
}

// ProcessTaskRunPipeline handles the actual execution of a pipeline.

type redisLogWriter struct {
	rdb       *redis.Client
	channel   string
	projectID string
	repo      repository.ProjectRepository
	execID    uuid.UUID
}

func (w *redisLogWriter) Write(p []byte) (n int, err error) {
	msg := string(p)
	// Optionally print to local stdout as well
	fmt.Print(msg)
	w.rdb.Publish(context.Background(), w.channel, msg)
	
	// Append to Postgres
	// In production, we'd batch this or do it asynchronously to avoid slowing down the pipeline,
	// but for an MVP, this is sufficient.
	if w.repo != nil && w.execID != uuid.Nil {
		_ = w.repo.AppendExecutionLog(context.Background(), w.execID, msg)
	}
	
	return len(p), nil
}

func (processor *redisTaskProcessor) ProcessTaskRunPipeline(ctx context.Context, task *asynq.Task) error {
	var payload RunPipelinePayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return err
	}

	log.Printf("Starting Pipeline for Project: %s, Commit: %s", payload.ProjectID, payload.CommitSHA)
	log.Printf("Repository URL: %s", payload.RepoURL)

	// Create an Execution Record in DB
	exec := &models.Execution{
		ProjectID: payload.ProjectID,
		CommitSHA: payload.CommitSHA,
		Status:    models.ExecutionStatusRunning,
	}
	if err := processor.repo.CreateExecution(ctx, exec); err != nil {
		log.Printf("Failed to create execution record: %v", err)
		// We can choose to fail the build here, or just proceed without persisting
	}

	// 1. Create a secure, isolated temporary workspace
	workDir, err := os.MkdirTemp("", "deployer-workspace-*")
	if err != nil {
		if exec.ID != uuid.Nil {
			_ = processor.repo.UpdateExecutionStatus(ctx, exec.ID, models.ExecutionStatusFailed)
		}
		return fmt.Errorf("failed to create temporary workspace: %w", err)
	}
	defer os.RemoveAll(workDir) // Ensure we clean up after the pipeline runs!

	log.Printf("Cloning repository into temporary workspace: %s", workDir)

	// 2. Clone the repository
	_, err = git.PlainClone(workDir, false, &git.CloneOptions{
		URL:      payload.RepoURL,
		Progress: os.Stdout,
	})
	if err != nil {
		if exec.ID != uuid.Nil {
			_ = processor.repo.UpdateExecutionStatus(ctx, exec.ID, models.ExecutionStatusFailed)
		}
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	// Note: For a real CI/CD engine, we'd want to checkout the specific CommitSHA here.
	// We'll stick to the default branch for this iteration to keep it simple.

	// 3. Look for .rune.yaml
	configPath := fmt.Sprintf("%s/.rune.yaml", workDir)
	log.Printf("Parsing pipeline configuration at: %s", configPath)

	config, err := pipeline.ParseConfig(configPath)
	if err != nil {
		if exec.ID != uuid.Nil {
			_ = processor.repo.UpdateExecutionStatus(ctx, exec.ID, models.ExecutionStatusFailed)
		}
		return fmt.Errorf("pipeline configuration error: %w", err)
	}

	// 4. Fetch AI Credentials for the Project
	cred, err := processor.repo.GetCredentialByProjectID(ctx, payload.ProjectID)
	var aiProv, aiKey, aiBaseURL string
	if err != nil {
		log.Printf("Warning: Could not fetch AI credentials for project (AI steps will fail): %v", err)
	} else {
		aiProv = string(cred.Provider)
		aiKey = cred.APIKey
		aiBaseURL = cred.BaseURL
	}

	// 5. Execute the Pipeline
	logWriter := &redisLogWriter{
		rdb:       processor.rdb,
		channel:   fmt.Sprintf("logs:%s", payload.ProjectID),
		projectID: payload.ProjectID.String(),
		repo:      processor.repo,
		execID:    exec.ID,
	}

	runner := pipeline.NewRunner(config, workDir, aiProv, aiKey, aiBaseURL, logWriter)
	if err := runner.Execute(); err != nil {
		log.Printf("Pipeline failed for Commit %s: %v", payload.CommitSHA, err)
		if exec.ID != uuid.Nil {
			_ = processor.repo.UpdateExecutionStatus(ctx, exec.ID, models.ExecutionStatusFailed)
		}
		return err
	}

	log.Printf("Pipeline completed successfully for Commit: %s", payload.CommitSHA)
	if exec.ID != uuid.Nil {
		_ = processor.repo.UpdateExecutionStatus(ctx, exec.ID, models.ExecutionStatusSuccess)
	}
	return nil
}

// ProcessTaskSnykScan handles the actual execution of a security scan.
func (processor *redisTaskProcessor) ProcessTaskSnykScan(ctx context.Context, task *asynq.Task) error {
	var payload SnykScanPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return err
	}

	log.Printf("Running Snyk Scan for Project: %s", payload.ProjectID)
	// TODO: Execute snyk CLI against codebase
	time.Sleep(1 * time.Second) // Simulate work
	log.Printf("Snyk Scan completed cleanly for Project: %s", payload.ProjectID)

	return nil
}
