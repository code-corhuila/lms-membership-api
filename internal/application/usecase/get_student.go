package usecase

import (
	"context"

	"github.com/code-corhuila/lms-membership-api/internal/domain/membership"
)

// GetStudent exposes a single student by ID over HTTP — needed by
// circulation-service to check eligibility (suspension status) before
// registering a loan, now that it can no longer query the `students` table
// directly (library-docs/09-microservices/service-boundary-rules.md).
type GetStudent struct {
	Students membership.StudentRepository
}

func NewGetStudent(students membership.StudentRepository) *GetStudent {
	return &GetStudent{Students: students}
}

func (uc *GetStudent) Execute(ctx context.Context, id string) (*membership.Student, error) {
	return uc.Students.FindByID(ctx, id)
}
