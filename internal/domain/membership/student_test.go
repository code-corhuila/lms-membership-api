package membership_test

import (
	"testing"

	"github.com/code-corhuila/lms-membership-api/internal/domain/membership"
	"github.com/code-corhuila/lms-membership-api/internal/domain/shared"
)

func mustEmail(t *testing.T, raw string) shared.Email {
	t.Helper()
	e, err := shared.NewEmail(raw)
	if err != nil {
		t.Fatalf("unexpected error creating email: %v", err)
	}
	return e
}

func TestNewStudent_IsEligibleForLoanByDefault(t *testing.T) {
	email := mustEmail(t, "student@example.com")
	s, err := membership.NewStudent("Jane Doe", "1075300000", email, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !s.IsEligibleForLoan() {
		t.Fatal("expected a newly registered student to be eligible for a loan")
	}
}

func TestSuspend_MakesStudentIneligibleForLoan(t *testing.T) {
	email := mustEmail(t, "student@example.com")
	s, _ := membership.NewStudent("Jane Doe", "1075300000", email, "")

	s.Suspend(7)

	if s.IsEligibleForLoan() {
		t.Fatal("expected a suspended student to be ineligible for a loan")
	}
}

func TestDeactivate_RejectsWhenHasActiveLoans(t *testing.T) {
	// INV-002: a student with active loans or an active suspension cannot be deactivated.
	email := mustEmail(t, "student@example.com")
	s, _ := membership.NewStudent("Jane Doe", "1075300000", email, "")

	if err := s.Deactivate(true); err != membership.ErrStudentHasActiveLoansOrSuspension {
		t.Fatalf("expected ErrStudentHasActiveLoansOrSuspension, got %v", err)
	}
}

func TestDeactivate_RejectsWhenSuspended(t *testing.T) {
	email := mustEmail(t, "student@example.com")
	s, _ := membership.NewStudent("Jane Doe", "1075300000", email, "")
	s.Suspend(7)

	if err := s.Deactivate(false); err != membership.ErrStudentHasActiveLoansOrSuspension {
		t.Fatalf("expected ErrStudentHasActiveLoansOrSuspension, got %v", err)
	}
}

func TestDeactivate_SucceedsWhenEligible(t *testing.T) {
	email := mustEmail(t, "student@example.com")
	s, _ := membership.NewStudent("Jane Doe", "1075300000", email, "")

	if err := s.Deactivate(false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.IsActive() {
		t.Fatal("expected student to be inactive after deactivation")
	}
}
