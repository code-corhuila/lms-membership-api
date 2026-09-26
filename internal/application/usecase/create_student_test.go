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

// fakeIdempotencyStore — Fake test double (11-quality/tdd-guide.md).
type fakeIdempotencyStore struct {
	byKey map[string]string
}

func newFakeIdempotencyStore() *fakeIdempotencyStore {
	return &fakeIdempotencyStore{byKey: map[string]string{}}
}

func (f *fakeIdempotencyStore) Get(_ context.Context, key string) (string, bool, error) {
	id, ok := f.byKey[key]
	return id, ok, nil
}

func (f *fakeIdempotencyStore) Save(_ context.Context, key, studentID string) error {
	f.byKey[key] = studentID
	return nil
}

func TestCreateStudent_Succeeds(t *testing.T) {
	repo := newFakeStudentRepository()
	uc := usecase.NewCreateStudent(repo, newFakeIdempotencyStore())

	student, replayed, err := uc.Execute(context.Background(), "Jane Doe", "1075300000", "jane@example.com", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if replayed {
		t.Fatal("first registration must not be reported as replayed")
	}
	if student.ID == "" {
		t.Fatal("expected the student to be assigned an ID")
	}
}

func TestCreateStudent_RejectsDuplicateDocumentID(t *testing.T) {
	// FR-004: reject registration when the document ID is already registered.
	repo := newFakeStudentRepository()
	uc := usecase.NewCreateStudent(repo, newFakeIdempotencyStore())

	_, _, err := uc.Execute(context.Background(), "Jane Doe", "1075300000", "jane@example.com", "", "")
	if err != nil {
		t.Fatalf("unexpected error on first registration: %v", err)
	}

	_, _, err = uc.Execute(context.Background(), "Someone Else", "1075300000", "other@example.com", "", "")
	if err != usecase.ErrDocumentIDAlreadyExists {
		t.Fatalf("expected ErrDocumentIDAlreadyExists, got %v", err)
	}
}

func TestCreateStudent_RepeatedIdempotencyKeyReplaysTheOriginal(t *testing.T) {
	// rules/2-anexos/C-api-hexagonal.md, numeral 5.3.8: a retry carrying the
	// same Idempotency-Key must return the original student, not a new one.
	repo := newFakeStudentRepository()
	uc := usecase.NewCreateStudent(repo, newFakeIdempotencyStore())

	first, replayed, err := uc.Execute(context.Background(), "Jane Doe", "1075300000", "jane@example.com", "", "retry-key-1")
	if err != nil {
		t.Fatalf("unexpected error on first registration: %v", err)
	}
	if replayed {
		t.Fatal("first registration must not be reported as replayed")
	}

	second, replayed, err := uc.Execute(context.Background(), "Jane Doe", "1075300000", "jane@example.com", "", "retry-key-1")
	if err != nil {
		t.Fatalf("unexpected error on retried registration: %v", err)
	}
	if !replayed {
		t.Fatal("expected the retry to be reported as replayed")
	}
	if second.ID != first.ID {
		t.Fatalf("expected the retry to return the original student %s, got %s", first.ID, second.ID)
	}
}
