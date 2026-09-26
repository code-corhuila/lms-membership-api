// Package out holds the driven (outbound) ports every use case depends on —
// what the application needs from the outside world, never how it's implemented
// (rules/2-anexos/C-api-hexagonal.md, "Puertos de salida").
package out

import (
	"context"

	"github.com/code-corhuila/lms-membership-api/internal/domain/membership"
)

// StudentRepository is the driven port for Student persistence.
type StudentRepository interface {
	FindByID(ctx context.Context, id string) (*membership.Student, error)
	FindByDocumentID(ctx context.Context, documentID string) (*membership.Student, error)
	Search(ctx context.Context, query string, page, limit int) (students []*membership.Student, total int, err error)
	Save(ctx context.Context, s *membership.Student) error
}

// ActiveLoansChecker is a driven port for asking circulation-service whether a
// student currently has active loans. Membership cannot query the `loans`
// table directly anymore — it lives in circulation-service's own database
// (library-docs/09-microservices/service-boundary-rules.md) — so this is
// implemented by an HTTP adapter, not a repository.
type ActiveLoansChecker interface {
	CountActive(ctx context.Context, studentID string) (int, error)
}

// IdempotencyStore is the driven port for the idempotent-creation check
// (rules/2-anexos/C-api-hexagonal.md, numeral 5.3.8): a repeated POST /students
// with the same Idempotency-Key must return the original student, not create a
// second one.
//
// Provisional: the only adapter today (internal/adapter/out/idempotency) is an
// in-memory map — it does not survive a restart, and does not coordinate
// across more than one running instance. A durable adapter belongs in
// lms-membership-db's own idempotency_key table (rules/2-anexos/A-db-postgres.md),
// which does not exist yet.
type IdempotencyStore interface {
	// Get returns the studentID previously saved under key, and found=true, or
	// found=false if key has never been used.
	Get(ctx context.Context, key string) (studentID string, found bool, err error)
	// Save records that key produced studentID. Must be a no-op error (not a
	// conflict) if key is saved again with the same studentID it already has.
	Save(ctx context.Context, key, studentID string) error
}
