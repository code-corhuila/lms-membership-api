package usecase

import (
	"context"
	"errors"

	"github.com/code-corhuila/lms-membership-api/internal/domain/membership"
	"github.com/code-corhuila/lms-membership-api/internal/domain/shared"
)

// ErrDocumentIDAlreadyExists — FR-004: rejects registration when the document ID
// is already registered.
var ErrDocumentIDAlreadyExists = errors.New("document id already exists")

// CreateStudent implements HU-02's acceptance criteria.
type CreateStudent struct {
	Students membership.StudentRepository
}

func NewCreateStudent(students membership.StudentRepository) *CreateStudent {
	return &CreateStudent{Students: students}
}

func (uc *CreateStudent) Execute(ctx context.Context, fullName, documentID, email, phone string) (*membership.Student, error) {
	if _, err := uc.Students.FindByDocumentID(ctx, documentID); err == nil {
		return nil, ErrDocumentIDAlreadyExists
	} else if !errors.Is(err, membership.ErrStudentNotFound) {
		return nil, err
	}

	parsedEmail, err := shared.NewEmail(email)
	if err != nil {
		return nil, err
	}

	student, err := membership.NewStudent(fullName, documentID, parsedEmail, phone)
	if err != nil {
		return nil, err
	}

	if err := uc.Students.Save(ctx, student); err != nil {
		return nil, err
	}
	return student, nil
}
