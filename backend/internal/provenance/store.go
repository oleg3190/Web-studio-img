package provenance

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Event struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	ProjectID   uuid.UUID
	IterationID *uuid.UUID
	EntityType  string
	EntityID    uuid.UUID
	Action      string
	Payload     map[string]any
	ParentHash  string
	Hash        string
	CreatedAt   time.Time
}

type Store struct{ db *sql.DB }

func NewStore(db *sql.DB) (*Store, error) {
	if db == nil {
		return nil, errors.New("provenance store requires database")
	}
	return &Store{db: db}, nil
}

func (s *Store) Append(ctx context.Context, event Event) (Event, error) {
	payload, err := json.Marshal(event.Payload)
	if err != nil {
		return Event{}, fmt.Errorf("marshal provenance payload: %w", err)
	}
	var parent string
	_ = s.db.QueryRowContext(ctx, `SELECT hash FROM provenance_events WHERE user_id=$1 ORDER BY sequence DESC LIMIT 1`, event.UserID).Scan(&parent)
	event.ParentHash = parent
	canonical := fmt.Sprintf("%s|%s|%s|%s|%s|%s", event.EntityType, event.EntityID, event.Action, string(payload), event.ParentHash, event.CreatedAt.UTC().Format(time.RFC3339Nano))
	sum := sha256.Sum256([]byte(canonical))
	event.Hash = hex.EncodeToString(sum[:])
	event.ID = uuid.New()
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO provenance_events
			(id, user_id, project_id, iteration_id, entity_type, entity_id, action, payload, parent_hash, hash, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
	`, event.ID, event.UserID, event.ProjectID, event.IterationID, event.EntityType, event.EntityID, event.Action, payload, nullableString(event.ParentHash), event.Hash, event.CreatedAt)
	if err != nil {
		return Event{}, fmt.Errorf("append provenance event: %w", err)
	}
	return event, nil
}

func nullableString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
