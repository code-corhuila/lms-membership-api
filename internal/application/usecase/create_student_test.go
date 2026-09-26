package usecase_test

import (
	"context"
	"testing"

	"github.com/code-corhuila/lms-membership-api/internal/application/usecase"
	"github.com/code-corhuila/lms-membership-api/internal/domain/membership"
)

// fakeStudentRepository — Fake test double (11-quality/tdd-guide.md).
type fakeStudentRepository struct {
	byDocumentID map[string]*membership.Student
	byID         map[string]*membership.Student
}

func newFakeStudentRepository() *fakeStudentRepository {
	return &fakeStudentRepository{
		byDocumentID: map[string]*membership.Student{},
		byID:         map[string]*membership.Student{},
	}
}

func (f *fakeStudentRepository) FindByID(_ context.Context, id string) (*membership.Student, error) {
	s, ok := f.byID[id]
	if !ok {
		return nil, membership.ErrStudentNotFound
	}
	return s, nil
}

func (f *fakeStudentRepository) FindByDocumentID(_ context.Context, documentID string) (*membership.Student, error) {
	s, ok := f.byDocumentID[documentID]
	if !ok {
		return nil, membership.ErrStudentNotFound
	}
	return s, nil
}

func (f *fakeStudentRepository) Search(_ context.Context, _ string, _, _ int) ([]*membership.Student, int, error) {
	return nil, 0, nil
}

func (f *fakeStudentRepository) Save(_ context.Context, s *membership.Student) error {
	f.byDocumentID[s.DocumentID] = s
	f.byID[s.ID] = s
	return nil
}

func TestCreateStudent_Succeeds(t *testing.T) {
	repo := newFakeStudentRepository()
	uc := usecase.NewCreateStudent(repo)

	student, err := uc.Execute(context.Background(), "Jane Doe", "1075300000", "jane@example.com", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if student.ID == "" {
		t.Fatal("expected the student to be assigned an ID")
	}
}

func TestCreateStudent_RejectsDuplicateDocumentID(t *testing.T) {
	// FR-004: reject registration when the document ID is already registered.
	repo := newFakeStudentRepository()
	uc := usecase.NewCreateStudent(repo)

	_, err := uc.Execute(context.Background(), "Jane Doe", "1075300000", "jane@example.com", "")
	if err != nil {
		t.Fatalf("unexpected error on first registration: %v", err)
	}

	_, err = uc.Execute(context.Background(), "Someone Else", "1075300000", "other@example.com", "")
	if err != usecase.ErrDocumentIDAlreadyExists {
		t.Fatalf("expected ErrDocumentIDAlreadyExists, got %v", err)
	}
}
