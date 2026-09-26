package usecase

import (
	"context"

	out "github.com/code-corhuila/lms-membership-api/internal/application/port/out"
	"github.com/code-corhuila/lms-membership-api/internal/domain/membership"
)

// SearchStudents implements the list/search half of HU-03.
type SearchStudents struct {
	Students out.StudentRepository
}

func NewSearchStudents(students out.StudentRepository) *SearchStudents {
	return &SearchStudents{Students: students}
}

func (uc *SearchStudents) Execute(ctx context.Context, query string, page, limit int) ([]*membership.Student, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	return uc.Students.Search(ctx, query, page, limit)
}
