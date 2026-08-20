package worker

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
)

const (
	TaskTypeRunPipeline = "pipeline:run"
	TaskTypeSnykScan    = "security:snyk_scan"
)

// RunPipelinePayload is the data required to run a full pipeline.
type RunPipelinePayload struct {
	ProjectID uuid.UUID `json:"project_id"`
	CommitSHA string    `json:"commit_sha"`
	RepoURL   string    `json:"repo_url"`
}

// SnykScanPayload is the data required to run a security scan.
type SnykScanPayload struct {
	ProjectID uuid.UUID `json:"project_id"`
	CommitSHA string    `json:"commit_sha"`
}

// TaskDistributor is an interface for enqueuing tasks.
type TaskDistributor interface {
	DistributeTaskRunPipeline(ctx context.Context, payload *RunPipelinePayload, opts ...asynq.Option) error
	DistributeTaskSnykScan(ctx context.Context, payload *SnykScanPayload, opts ...asynq.Option) error
}

type redisTaskDistributor struct {
	client *asynq.Client
}

func NewRedisTaskDistributor(redisOpt asynq.RedisConnOpt) TaskDistributor {
	client := asynq.NewClient(redisOpt)
	return &redisTaskDistributor{
		client: client,
	}
}

func (distributor *redisTaskDistributor) DistributeTaskRunPipeline(ctx context.Context, payload *RunPipelinePayload, opts ...asynq.Option) error {
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal task payload: %w", err)
	}

	task := asynq.NewTask(TaskTypeRunPipeline, jsonPayload, opts...)
	info, err := distributor.client.EnqueueContext(ctx, task)
	if err != nil {
		return fmt.Errorf("failed to enqueue task: %w", err)
	}

	fmt.Printf("Enqueued task: type=%s, id=%s, queue=%s\n", task.Type(), info.ID, info.Queue)
	return nil
}

func (distributor *redisTaskDistributor) DistributeTaskSnykScan(ctx context.Context, payload *SnykScanPayload, opts ...asynq.Option) error {
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal task payload: %w", err)
	}

	task := asynq.NewTask(TaskTypeSnykScan, jsonPayload, opts...)
	info, err := distributor.client.EnqueueContext(ctx, task)
	if err != nil {
		return fmt.Errorf("failed to enqueue task: %w", err)
	}

	fmt.Printf("Enqueued task: type=%s, id=%s, queue=%s\n", task.Type(), info.ID, info.Queue)
	return nil
}
