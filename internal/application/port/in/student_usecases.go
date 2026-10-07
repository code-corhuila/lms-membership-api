// Package in holds the driving (inbound) ports — what this service offers,
// as interfaces the HTTP adapter depends on instead of the concrete use case
// structs directly (rules/2-anexos/C-api-hexagonal.md, "Puertos de entrada").
package in

import (
	"context"

	"github.com/code-corhuila/lms-membership-api/internal/domain/membership"
)

// CreateStudentUseCase registers a new Student (HU-02).
//
// replayed reports whether idempotencyKey had already been used: when true,
// the returned student is the one originally created by that key, and the
// HTTP adapter must answer 200, not 201 (rules/2-anexos/C-api-hexagonal.md,
// numeral 5.3.8).
type CreateStudentUseCase interface {
	Execute(ctx context.Context, fullName, documentID, email, phone, idempotencyKey string) (student *membership.Student, replayed bool, err error)
}

// GetStudentUseCase reads a single Student by id.
type GetStudentUseCase interface {
	Execute(ctx context.Context, id string) (*membership.Student, error)
}

// UpdateStudentUseCase edits a Student's contact information (HU-03, Scenario 1).
type UpdateStudentUseCase interface {
	Execute(ctx context.Context, id, fullName, email, phone string) (*membership.Student, error)
}

// DeactivateStudentUseCase soft-deletes a Student (HU-03, Scenario 2).
type DeactivateStudentUseCase interface {
	Execute(ctx context.Context, id string) (*membership.Student, error)
}

// SearchStudentsUseCase lists Students with a bounded, offset-paginated search (HU-03).
type SearchStudentsUseCase interface {
	Execute(ctx context.Context, query string, page, limit int) (students []*membership.Student, total int, err error)
}

// SuspendStudentUseCase applies a fixed suspension window, called by
// circulation-service on a late return (HU-08, INV-006).
type SuspendStudentUseCase interface {
	Execute(ctx context.Context, id string, days int) (*membership.Student, error)
}
