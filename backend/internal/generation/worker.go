package generation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"

	"github.com/oleg3190/Web-studio-img/backend/internal/assets"
	"github.com/oleg3190/Web-studio-img/backend/internal/provenance"
)

const TaskType = "generation:execute"

type TaskPayload struct {
	UserID       uuid.UUID `json:"user_id"`
	GenerationID uuid.UUID `json:"generation_id"`
}

func NewTask(userID, generationID uuid.UUID) (*asynq.Task, error) {
	payload, err := json.Marshal(TaskPayload{UserID: userID, GenerationID: generationID})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TaskType, payload), nil
}

type Worker struct {
	Store    Store
	Provider Provider
	Assets   *assets.Processor
	Provenance *provenance.Store
}

func (w *Worker) Handle(ctx context.Context, task *asynq.Task) error {
	var payload TaskPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("decode generation task: %w", err)
	}
	item, err := w.Store.GetOwned(ctx, payload.UserID, payload.GenerationID)
	if err != nil {
		return err
	}
	if item.Status == StatusSucceeded || item.Status == StatusCancelled {
		return nil
	}
	now := time.Now()
	if err := w.Store.MarkRunning(ctx, payload.UserID, item.ID, now); err != nil && !errors.Is(err, ErrGenerationNotFound) {
		return err
	}
	result, err := w.Provider.Generate(ctx, Request{
		ProjectID: item.ProjectID, IterationID: item.IterationID, Prompt: item.Prompt,
		NegativePrompt: item.NegativePrompt, Seed: item.Seed, AspectRatio: item.AspectRatio,
		Parameters: item.Parameters, IdempotencyKey: payload.GenerationID.String(),
	})
	if err != nil {
		code := "provider_error"
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			code = "provider_timeout"
		}
		if !isRetryable(err) {
			_ = w.Store.MarkFailed(ctx, payload.UserID, item.ID, code, err.Error(), time.Now())
			if w.Provenance != nil { _, _ = w.Provenance.Append(ctx, provenance.Event{UserID: payload.UserID, ProjectID: item.ProjectID, IterationID: item.IterationID, EntityType: "generation", EntityID: item.ID, Action: "generation_failed", Payload: map[string]any{"code": code, "message": err.Error()}, CreatedAt: time.Now()}) }
			return nil
		}
		return err
	}
	for _, image := range result.Images {
		if _, err := w.Assets.Process(ctx, payload.UserID, item.ProjectID, item.ID, image.Data, image.ContentType); err != nil {
			_ = w.Store.MarkFailed(ctx, payload.UserID, item.ID, "asset_error", err.Error(), time.Now())
			return nil
		}
	}
	var version *string
	if result.ModelVersion != "" {
		version = &result.ModelVersion
	}
	var jobID *string
	if result.ProviderJobID != "" {
		jobID = &result.ProviderJobID
	}
	if err := w.Store.MarkSucceeded(ctx, payload.UserID, item.ID, version, jobID, result.Cost, time.Now()); err != nil {
		return err
	}
	if w.Provenance != nil {
		_, _ = w.Provenance.Append(ctx, provenance.Event{UserID: payload.UserID, ProjectID: item.ProjectID, IterationID: item.IterationID, EntityType: "generation", EntityID: item.ID, Action: "generation_succeeded", Payload: map[string]any{"provider": item.Provider, "model": item.Model, "model_version": result.ModelVersion, "provider_job_id": result.ProviderJobID, "cost": result.Cost}, CreatedAt: time.Now()})
	}
	return nil
}

func isRetryable(err error) bool {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var retryable interface{ Retryable() bool }
	return errors.As(err, &retryable) && retryable.Retryable()
}
