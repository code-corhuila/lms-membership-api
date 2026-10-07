// Package membership implements the Membership bounded context
// (library-docs/02-domain/domain-map.md): student records the Administrator manages.
// A Student is never a system user — see library-docs/01-context/overview.md.
package membership

import (
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/code-corhuila/lms-membership-api/internal/domain/shared"
)

// ErrStudentHasActiveLoansOrSuspension — INV-002: a student with active loans or an
// active suspension cannot be deactivated (library-docs/02-domain/entities-and-rules.md).
var ErrStudentHasActiveLoansOrSuspension = errors.New("student has active loans or an active suspension")

// ErrStudentNotFound is the port-level "not found" error (see AdministratorRepository
// for why use cases depend on this, not an infrastructure error type).
var ErrStudentNotFound = errors.New("student not found")

// Student is the Membership bounded context's Aggregate Root.
type Student struct {
	ID             string
	FullName       string
	DocumentID     string
	Email          shared.Email
	Phone          string // optional
	SuspendedUntil *time.Time
	DeactivatedAt  *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// NewStudent constructs a Student for registration (HU-02).
func NewStudent(fullName, documentID string, email shared.Email, phone string) (*Student, error) {
	if fullName == "" {
		return nil, errors.New("fullName must not be empty")
	}
	if documentID == "" {
		return nil, errors.New("documentId must not be empty")
	}

	now := time.Now().UTC()
	return &Student{
		ID:         uuid.NewString(),
		FullName:   fullName,
		DocumentID: documentID,
		Email:      email,
		Phone:      phone,
		CreatedAt:  now,
		UpdatedAt:  now,
	}, nil
}

// IsEligibleForLoan returns true if the student is not currently suspended.
func (s *Student) IsEligibleForLoan() bool {
	return s.SuspendedUntil == nil || s.SuspendedUntil.Before(time.Now().UTC())
}

// Suspend applies a flat suspension of `days` from now — INV-006 on Loan
// (library-docs/02-domain/entities-and-rules.md): late return triggers a fixed 7-day
// suspension, not proportional to days late.
func (s *Student) Suspend(days int) {
	until := time.Now().UTC().AddDate(0, 0, days)
	s.SuspendedUntil = &until
	s.UpdatedAt = time.Now().UTC()
}

// IsActive returns true if the student has not been deactivated.
func (s *Student) IsActive() bool {
	return s.DeactivatedAt == nil
}

// Update applies an edit to contact information (HU-03, Scenario 1). documentId is
// intentionally not editable through this method — it is the immutable business key.
func (s *Student) Update(fullName string, email shared.Email, phone string) error {
	if fullName == "" {
		return errors.New("fullName must not be empty")
	}
	s.FullName = fullName
	s.Email = email
	s.Phone = phone
	s.UpdatedAt = time.Now().UTC()
	return nil
}

// Deactivate soft-deletes the student (HU-03). `hasActiveLoans` must be computed by
// the caller (a domain service) — Student cannot know about Loan directly, since Loan
// belongs to the Circulation bounded context.
func (s *Student) Deactivate(hasActiveLoans bool) error {
	if hasActiveLoans || !s.IsEligibleForLoan() {
		return ErrStudentHasActiveLoansOrSuspension
	}
	now := time.Now().UTC()
	s.DeactivatedAt = &now
	s.UpdatedAt = now
	return nil
}
