package usecase

import (
	"context"
	"errors"

	out "github.com/code-corhuila/lms-membership-api/internal/application/port/out"
	"github.com/code-corhuila/lms-membership-api/internal/domain/membership"
	"github.com/code-corhuila/lms-membership-api/internal/domain/shared"
)

// ErrDocumentIDAlreadyExists — FR-004: rejects registration when the document ID
// is already registered.
var ErrDocumentIDAlreadyExists = errors.New("document id already exists")

// CreateStudent implements HU-02's acceptance criteria, plus the idempotent
// creation rules/2-anexos/C-api-hexagonal.md (numeral 5.3.8) requires: a
// repeated request carrying the same Idempotency-Key must not create a
// second Student.
type CreateStudent struct {
	Students    out.StudentRepository
	Idempotency out.IdempotencyStore
}

func NewCreateStudent(students out.StudentRepository, idempotency out.IdempotencyStore) *CreateStudent {
	return &CreateStudent{Students: students, Idempotency: idempotency}
}

// Execute registers a Student. replayed=true means idempotencyKey had
// already been used — the returned student is the original one, and the
// caller (the HTTP adapter) must answer 200, not 201.
func (uc *CreateStudent) Execute(ctx context.Context, fullName, documentID, email, phone, idempotencyKey string) (student *membership.Student, replayed bool, err error) {
	if idempotencyKey != "" {
		if existingID, found, err := uc.Idempotency.Get(ctx, idempotencyKey); err != nil {
			return nil, false, err
		} else if found {
			existing, err := uc.Students.FindByID(ctx, existingID)
			if err != nil {
				return nil, false, err
			}
			return existing, true, nil
		}
	}

	if _, err := uc.Students.FindByDocumentID(ctx, documentID); err == nil {
		return nil, false, ErrDocumentIDAlreadyExists
	} else if !errors.Is(err, membership.ErrStudentNotFound) {
		return nil, false, err
	}

	parsedEmail, err := shared.NewEmail(email)
	if err != nil {
		return nil, false, err
	}

	newStudent, err := membership.NewStudent(fullName, documentID, parsedEmail, phone)
	if err != nil {
		return nil, false, err
	}

	if err := uc.Students.Save(ctx, newStudent); err != nil {
		return nil, false, err
	}

	if idempotencyKey != "" {
		if err := uc.Idempotency.Save(ctx, idempotencyKey, newStudent.ID); err != nil {
			return nil, false, err
		}
	}

	return newStudent, false, nil
}
