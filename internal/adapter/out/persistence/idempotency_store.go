package persistence

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// IdempotencyStore implements out.IdempotencyStore against PostgreSQL's
// membership.idempotency_key table (lms-membership-db). Durable — replaces
// the provisional in-memory adapter now that the table exists.
type IdempotencyStore struct {
	db *pgxpool.Pool
}

func NewIdempotencyStore(db *pgxpool.Pool) *IdempotencyStore {
	return &IdempotencyStore{db: db}
}

func (s *IdempotencyStore) Get(ctx context.Context, key string) (studentID string, found bool, err error) {
	const query = `SELECT student_id FROM membership.idempotency_key WHERE key = $1`
	err = s.db.QueryRow(ctx, query, key).Scan(&studentID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return studentID, true, nil
}

// Save is an upsert on the same key/studentID pair — CreateStudent only ever
// calls Save once per key (after a fresh registration, never on a replay),
// but ON CONFLICT DO NOTHING keeps this safe if it's ever called twice.
func (s *IdempotencyStore) Save(ctx context.Context, key, studentID string) error {
	const query = `
		INSERT INTO membership.idempotency_key (key, student_id)
		VALUES ($1, $2)
		ON CONFLICT (key) DO NOTHING`
	_, err := s.db.Exec(ctx, query, key, studentID)
	return err
}
