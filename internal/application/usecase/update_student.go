package usecase

import (
	"context"

	"github.com/code-corhuila/lms-membership-api/internal/domain/membership"
	"github.com/code-corhuila/lms-membership-api/internal/domain/shared"
)

// UpdateStudent implements HU-03, Scenario 1 (edit contact information).
type UpdateStudent struct {
	Students membership.StudentRepository
}

func NewUpdateStudent(students membership.StudentRepository) *UpdateStudent {
	return &UpdateStudent{Students: students}
}

func (uc *UpdateStudent) Execute(ctx context.Context, id, fullName, email, phone string) (*membership.Student, error) {
	student, err := uc.Students.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	parsedEmail, err := shared.NewEmail(email)
	if err != nil {
		return nil, err
	}

	if err := student.Update(fullName, parsedEmail, phone); err != nil {
		return nil, err
	}

	if err := uc.Students.Save(ctx, student); err != nil {
		return nil, err
	}
	return student, nil
}
