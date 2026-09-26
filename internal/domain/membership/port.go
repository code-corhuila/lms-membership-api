package membership

import "context"

// StudentRepository is the driven port for Student persistence.
type StudentRepository interface {
	FindByID(ctx context.Context, id string) (*Student, error)
	FindByDocumentID(ctx context.Context, documentID string) (*Student, error)
	Search(ctx context.Context, query string, page, limit int) (students []*Student, total int, err error)
	Save(ctx context.Context, s *Student) error
}
