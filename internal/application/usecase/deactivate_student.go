package usecase

import (
	"context"

	"github.com/code-corhuila/lms-membership-api/internal/domain/membership"
)

// DeactivateStudent implements HU-03, Scenario 2 — this is the one place in
// membership-service that needs Circulation's data, since a Student cannot
// know about its own active loans directly
// (library-docs/02-domain/entities-and-rules.md, Student.Deactivate). Before
// the microservices split this was an in-process repository call; now it's
// an HTTP call to circulation-service (membership.ActiveLoansChecker).
type DeactivateStudent struct {
	Students    membership.StudentRepository
	LoanChecker membership.ActiveLoansChecker
}

func NewDeactivateStudent(students membership.StudentRepository, loanChecker membership.ActiveLoansChecker) *DeactivateStudent {
	return &DeactivateStudent{Students: students, LoanChecker: loanChecker}
}

func (uc *DeactivateStudent) Execute(ctx context.Context, id string) (*membership.Student, error) {
	student, err := uc.Students.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	activeLoans, err := uc.LoanChecker.CountActive(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := student.Deactivate(activeLoans > 0); err != nil {
		return nil, err
	}

	if err := uc.Students.Save(ctx, student); err != nil {
		return nil, err
	}
	return student, nil
}
