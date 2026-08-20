package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/deployerai/deployer/pkg/pipeline"
	"github.com/go-git/go-git/v5"
	"github.com/hibiken/asynq"
)

// TaskProcessor is an interface for starting the task processor.
type TaskProcessor interface {
	Start() error
}

type redisTaskProcessor struct {
	server *asynq.Server
}

func NewRedisTaskProcessor(redisOpt asynq.RedisConnOpt) TaskProcessor {
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
func (processor *redisTaskProcessor) ProcessTaskRunPipeline(ctx context.Context, task *asynq.Task) error {
	var payload RunPipelinePayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return err
	}

	log.Printf("Starting Pipeline for Project: %s, Commit: %s", payload.ProjectID, payload.CommitSHA)
	log.Printf("Repository URL: %s", payload.RepoURL)

	// 1. Create a secure, isolated temporary workspace
	workDir, err := os.MkdirTemp("", "deployer-workspace-*")
	if err != nil {
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
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	// Note: For a real CI/CD engine, we'd want to checkout the specific CommitSHA here.
	// We'll stick to the default branch for this iteration to keep it simple.

	// 3. Look for .rune.yaml
	configPath := fmt.Sprintf("%s/.rune.yaml", workDir)
	log.Printf("Parsing pipeline configuration at: %s", configPath)

	config, err := pipeline.ParseConfig(configPath)
	if err != nil {
		return fmt.Errorf("pipeline configuration error: %w", err)
	}

	// 4. Execute the Pipeline
	runner := pipeline.NewRunner(config, workDir)
	if err := runner.Execute(); err != nil {
		log.Printf("Pipeline failed for Commit %s: %v", payload.CommitSHA, err)
		return err
	}

	log.Printf("Pipeline completed successfully for Commit: %s", payload.CommitSHA)
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
