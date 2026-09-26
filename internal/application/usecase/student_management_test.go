package usecase_test

import (
	"context"
	"testing"

	"github.com/code-corhuila/lms-membership-api/internal/application/usecase"
	"github.com/code-corhuila/lms-membership-api/internal/domain/membership"
	"github.com/code-corhuila/lms-membership-api/internal/domain/shared"
)

// fakeActiveLoansChecker — Fake test double (11-quality/tdd-guide.md) standing
// in for the HTTP call to circulation-service that DeactivateStudent makes.
type fakeActiveLoansChecker struct {
	activeCountByStudent map[string]int
}

func (f *fakeActiveLoansChecker) CountActive(_ context.Context, studentID string) (int, error) {
	return f.activeCountByStudent[studentID], nil
}

func seedFakeStudent(t *testing.T, repo *fakeStudentRepository) *membership.Student {
	t.Helper()
	email, err := shared.NewEmail("jane@example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	student, err := membership.NewStudent("Jane Doe", "1075300000", email, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := repo.Save(context.Background(), student); err != nil {
		t.Fatalf("unexpected error saving fixture: %v", err)
	}
	return student
}

func TestUpdateStudent_Succeeds(t *testing.T) {
	repo := newFakeStudentRepository()
	student := seedFakeStudent(t, repo)
	uc := usecase.NewUpdateStudent(repo)

	updated, err := uc.Execute(context.Background(), student.ID, "Jane A. Doe", "jane.new@example.com", "3000000000")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.FullName != "Jane A. Doe" {
		t.Fatalf("expected updated full name, got %q", updated.FullName)
	}
}

func TestUpdateStudent_FailsWhenNotFound(t *testing.T) {
	repo := newFakeStudentRepository()
	uc := usecase.NewUpdateStudent(repo)

	_, err := uc.Execute(context.Background(), "does-not-exist", "Name", "a@b.com", "")
	if err != membership.ErrStudentNotFound {
		t.Fatalf("expected ErrStudentNotFound, got %v", err)
	}
}

func TestDeactivateStudent_RejectsWhenHasActiveLoans(t *testing.T) {
	// INV-002 / FR-006: cannot deactivate a student with active loans.
	repo := newFakeStudentRepository()
	student := seedFakeStudent(t, repo)
	loans := &fakeActiveLoansChecker{activeCountByStudent: map[string]int{student.ID: 1}}
	uc := usecase.NewDeactivateStudent(repo, loans)

	_, err := uc.Execute(context.Background(), student.ID)
	if err != membership.ErrStudentHasActiveLoansOrSuspension {
		t.Fatalf("expected ErrStudentHasActiveLoansOrSuspension, got %v", err)
	}
}

func TestDeactivateStudent_SucceedsWithNoActiveLoans(t *testing.T) {
	repo := newFakeStudentRepository()
	student := seedFakeStudent(t, repo)
	loans := &fakeActiveLoansChecker{activeCountByStudent: map[string]int{}}
	uc := usecase.NewDeactivateStudent(repo, loans)

	deactivated, err := uc.Execute(context.Background(), student.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deactivated.IsActive() {
		t.Fatal("expected student to be deactivated")
	}
}
