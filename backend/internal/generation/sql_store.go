package generation

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type SQLStore struct{ db *sql.DB }

func NewSQLStore(db *sql.DB) (*SQLStore, error) {
	if db == nil {
		return nil, errors.New("generation store requires database")
	}
	return &SQLStore{db: db}, nil
}

func (s *SQLStore) Create(ctx context.Context, userID uuid.UUID, req Request, provider, model string) (Generation, bool, error) {
	if err := ValidateRequest(req); err != nil {
		return Generation{}, false, err
	}
	params, err := json.Marshal(req.Parameters)
	if err != nil {
		return Generation{}, false, fmt.Errorf("marshal generation parameters: %w", err)
	}
	var item Generation
	var raw []byte
	err = s.db.QueryRowContext(ctx, `
		WITH owned_project AS (
			SELECT id FROM projects WHERE id = $1 AND user_id = $2 AND status <> 'deleted'
		)
		INSERT INTO generations
			(user_id, project_id, iteration_id, provider, model, prompt, negative_prompt, seed, aspect_ratio, parameters, status, idempotency_key)
		SELECT $2, p.id, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
		FROM owned_project p
		ON CONFLICT (user_id, idempotency_key) DO NOTHING
		RETURNING id, project_id, iteration_id, user_id, provider, model, model_version, prompt, negative_prompt,
			seed, aspect_ratio, parameters, status, provider_job_id, error_code, error_message, cost, created_at, started_at, completed_at
	`, req.ProjectID, userID, req.IterationID, provider, model, req.Prompt, req.NegativePrompt, req.Seed, req.AspectRatio, params, StatusQueued, req.IdempotencyKey).Scan(
		&item.ID, &item.ProjectID, &item.IterationID, &item.UserID, &item.Provider, &item.Model, &item.ModelVersion,
		&item.Prompt, &item.NegativePrompt, &item.Seed, &item.AspectRatio, &raw, &item.Status, &item.ProviderJobID,
		&item.ErrorCode, &item.ErrorMessage, &item.Cost, &item.CreatedAt, &item.StartedAt, &item.CompletedAt,
	)
	if err == nil {
		_ = json.Unmarshal(raw, &item.Parameters)
		return item, true, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return Generation{}, false, fmt.Errorf("create generation: %w", err)
	}
	item, err = s.GetByIdempotency(ctx, userID, req.IdempotencyKey)
	if err != nil {
		return Generation{}, false, err
	}
	if item.ProjectID != req.ProjectID {
		return Generation{}, false, ErrIdempotencyConflict
	}
	return item, false, nil
}

func (s *SQLStore) GetByIdempotency(ctx context.Context, userID uuid.UUID, key string) (Generation, error) {
	return s.scanOne(ctx, `
		SELECT id, project_id, iteration_id, user_id, provider, model, model_version, prompt, negative_prompt,
			seed, aspect_ratio, parameters, status, provider_job_id, error_code, error_message, cost, created_at, started_at, completed_at
		FROM generations WHERE user_id = $1 AND idempotency_key = $2
	`, userID, key)
}

func (s *SQLStore) GetOwned(ctx context.Context, userID, id uuid.UUID) (Generation, error) {
	return s.scanOne(ctx, `
		SELECT g.id, g.project_id, g.iteration_id, g.user_id, g.provider, g.model, g.model_version, g.prompt,
			g.negative_prompt, g.seed, g.aspect_ratio, g.parameters, g.status, g.provider_job_id, g.error_code,
			g.error_message, g.cost, g.created_at, g.started_at, g.completed_at
		FROM generations g JOIN projects p ON p.id = g.project_id
		WHERE g.id = $1 AND g.user_id = $2 AND p.user_id = $2 AND p.status <> 'deleted'
	`, id, userID)
}

func (s *SQLStore) MarkRunning(ctx context.Context, userID, id uuid.UUID, at time.Time) error {
	res, err := s.db.ExecContext(ctx, `UPDATE generations SET status=$1, started_at=$2 WHERE id=$3 AND user_id=$4 AND status=$5`, StatusRunning, at, id, userID, StatusQueued)
	if err != nil {
		return fmt.Errorf("mark generation running: %w", err)
	}
	return expectOne(res, ErrGenerationNotFound)
}

func (s *SQLStore) MarkSucceeded(ctx context.Context, userID, id uuid.UUID, modelVersion, providerJobID *string, cost *float64, at time.Time) error {
	res, err := s.db.ExecContext(ctx, `UPDATE generations SET status=$1, model_version=$2, provider_job_id=$3, cost=$4, completed_at=$5 WHERE id=$6 AND user_id=$7 AND status IN ($8,$9)`, StatusSucceeded, modelVersion, providerJobID, cost, at, id, userID, StatusRunning, StatusQueued)
	if err != nil {
		return fmt.Errorf("mark generation succeeded: %w", err)
	}
	return expectOne(res, ErrGenerationNotFound)
}

func (s *SQLStore) MarkFailed(ctx context.Context, userID, id uuid.UUID, code, message string, at time.Time) error {
	res, err := s.db.ExecContext(ctx, `UPDATE generations SET status=$1, error_code=$2, error_message=$3, completed_at=$4 WHERE id=$5 AND user_id=$6 AND status IN ($7,$8)`, StatusFailed, code, message, at, id, userID, StatusRunning, StatusQueued)
	if err != nil {
		return fmt.Errorf("mark generation failed: %w", err)
	}
	return expectOne(res, ErrGenerationNotFound)
}

func (s *SQLStore) MarkCancelled(ctx context.Context, userID, id uuid.UUID, at time.Time) error {
	res, err := s.db.ExecContext(ctx, `UPDATE generations SET status=$1, completed_at=$2 WHERE id=$3 AND user_id=$4 AND status IN ($5,$6)`, StatusCancelled, at, id, userID, StatusQueued, StatusRunning)
	if err != nil {
		return fmt.Errorf("mark generation cancelled: %w", err)
	}
	return expectOne(res, ErrGenerationNotFound)
}

func expectOne(res sql.Result, notFound error) error {
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return notFound
	}
	return nil
}

func (s *SQLStore) scanOne(ctx context.Context, query string, args ...any) (Generation, error) {
	var item Generation
	var raw []byte
	err := s.db.QueryRowContext(ctx, query, args...).Scan(
		&item.ID, &item.ProjectID, &item.IterationID, &item.UserID, &item.Provider, &item.Model, &item.ModelVersion,
		&item.Prompt, &item.NegativePrompt, &item.Seed, &item.AspectRatio, &raw, &item.Status, &item.ProviderJobID,
		&item.ErrorCode, &item.ErrorMessage, &item.Cost, &item.CreatedAt, &item.StartedAt, &item.CompletedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Generation{}, ErrGenerationNotFound
	}
	if err != nil {
		return Generation{}, fmt.Errorf("get generation: %w", err)
	}
	if len(raw) > 0 && json.Unmarshal(raw, &item.Parameters) != nil {
		return Generation{}, fmt.Errorf("decode generation parameters")
	}
	return item, nil
}
