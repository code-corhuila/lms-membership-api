package usecase

import (
	"context"

	out "github.com/code-corhuila/lms-membership-api/internal/application/port/out"
	"github.com/code-corhuila/lms-membership-api/internal/domain/membership"
)

// SuspendStudent exposes Student.Suspend over HTTP — called by
// circulation-service when a return is late (HU-08, INV-006). Circulation
// decides *when* to suspend; Membership owns *how* (mutating and persisting
// its own aggregate).
type SuspendStudent struct {
	Students out.StudentRepository
}

func NewSuspendStudent(students out.StudentRepository) *SuspendStudent {
	return &SuspendStudent{Students: students}
}

func (uc *SuspendStudent) Execute(ctx context.Context, id string, days int) (*membership.Student, error) {
	student, err := uc.Students.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	student.Suspend(days)

	if err := uc.Students.Save(ctx, student); err != nil {
		return nil, err
	}
	return student, nil
}
