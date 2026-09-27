package generation

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusQueued Status = "queued"
	StatusRunning Status = "running"
	StatusSucceeded Status = "succeeded"
	StatusFailed Status = "failed"
	StatusCancelled Status = "cancelled"
)

var (
	ErrInvalidGeneration = errors.New("invalid generation")
	ErrGenerationNotFound = errors.New("generation not found")
	ErrIdempotencyConflict = errors.New("idempotency key belongs to another request")
)

type Request struct {
	ProjectID      uuid.UUID
	IterationID    *uuid.UUID
	Prompt         string
	NegativePrompt string
	Seed           *int64
	AspectRatio    string
	Parameters     map[string]any
	IdempotencyKey string
}

type Generation struct {
	ID             uuid.UUID      `json:"id"`
	ProjectID      uuid.UUID      `json:"project_id"`
	IterationID    *uuid.UUID     `json:"iteration_id,omitempty"`
	UserID         uuid.UUID      `json:"user_id"`
	Provider       string         `json:"provider"`
	Model          string         `json:"model"`
	ModelVersion   *string        `json:"model_version,omitempty"`
	Prompt         string         `json:"prompt"`
	NegativePrompt string         `json:"negative_prompt,omitempty"`
	Seed           *int64         `json:"seed,omitempty"`
	AspectRatio    string         `json:"aspect_ratio,omitempty"`
	Parameters     map[string]any `json:"parameters,omitempty"`
	Status         Status         `json:"status"`
	ProviderJobID  *string        `json:"provider_job_id,omitempty"`
	ErrorCode      *string        `json:"error_code,omitempty"`
	ErrorMessage   *string        `json:"error_message,omitempty"`
	Cost           *float64       `json:"cost,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	StartedAt      *time.Time     `json:"started_at,omitempty"`
	CompletedAt    *time.Time     `json:"completed_at,omitempty"`
}

type Image struct {
	Data        []byte
	ContentType string
	Filename    string
}

type ProviderResult struct {
	Images        []Image
	ModelVersion  string
	ProviderJobID string
	Cost          *float64
}

type Provider interface {
	Name() string
	Model() string
	Generate(context.Context, Request) (ProviderResult, error)
	Cancel(context.Context, string) error
}

type Store interface {
	Create(context.Context, uuid.UUID, Request, string, string) (Generation, bool, error)
	GetOwned(context.Context, uuid.UUID, uuid.UUID) (Generation, error)
	MarkRunning(context.Context, uuid.UUID, uuid.UUID, time.Time) error
	MarkSucceeded(context.Context, uuid.UUID, uuid.UUID, *string, *string, *float64, time.Time) error
	MarkFailed(context.Context, uuid.UUID, uuid.UUID, string, string, time.Time) error
	MarkCancelled(context.Context, uuid.UUID, uuid.UUID, time.Time) error
}

func ValidateRequest(r Request) error {
	if r.ProjectID == uuid.Nil || strings.TrimSpace(r.Prompt) == "" || len([]rune(r.Prompt)) > 20000 {
		return ErrInvalidGeneration
	}
	if len([]rune(r.NegativePrompt)) > 10000 || len(r.IdempotencyKey) > 200 {
		return ErrInvalidGeneration
	}
	if r.AspectRatio == "" {
		r.AspectRatio = "1:1"
	}
	if r.IdempotencyKey == "" {
		return ErrInvalidGeneration
	}
	return nil
}
