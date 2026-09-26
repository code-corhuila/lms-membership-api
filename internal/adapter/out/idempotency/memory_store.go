// Package idempotency implements application/port/out.IdempotencyStore.
//
// Provisional: this is an in-memory map, not the durable idempotency_key
// table rules/2-anexos/A-db-postgres.md describes — lms-membership-db does
// not have that table yet (ADR-010-liquibase-for-database-migrations.md
// tracks the -db restructuring this depends on). It does not survive a
// restart and does not coordinate across more than one running instance of
// this service. Replace with a Postgres-backed adapter once that table
// exists; the port (out.IdempotencyStore) does not change.
package idempotency

import (
	"context"
	"sync"
)

// MemoryStore is a process-local, mutex-guarded IdempotencyStore.
type MemoryStore struct {
	mu    sync.RWMutex
	byKey map[string]string
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{byKey: map[string]string{}}
}

func (s *MemoryStore) Get(_ context.Context, key string) (studentID string, found bool, err error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	studentID, found = s.byKey[key]
	return studentID, found, nil
}

func (s *MemoryStore) Save(_ context.Context, key, studentID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.byKey[key] = studentID
	return nil
}
