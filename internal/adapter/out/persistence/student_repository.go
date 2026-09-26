package persistence

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/code-corhuila/lms-membership-api/internal/domain/membership"
	"github.com/code-corhuila/lms-membership-api/internal/domain/shared"
)

// StudentRepository implements out.StudentRepository against
// PostgreSQL — the only code allowed to touch the `students` table
// (library-docs/09-microservices/service-boundary-rules.md).
type StudentRepository struct {
	db *pgxpool.Pool
}

func NewStudentRepository(db *pgxpool.Pool) *StudentRepository {
	return &StudentRepository{db: db}
}

const studentColumns = `id, full_name, document_id, email, COALESCE(phone, ''), suspended_until, deactivated_at, created_at, updated_at`

func scanStudent(row pgx.Row) (*membership.Student, error) {
	var s membership.Student
	var email string
	if err := row.Scan(&s.ID, &s.FullName, &s.DocumentID, &email, &s.Phone, &s.SuspendedUntil, &s.DeactivatedAt, &s.CreatedAt, &s.UpdatedAt); err != nil {
		return nil, err
	}
	// Already validated on the way in (NewStudent) — re-parsing here just
	// rehydrates the Value Object, it cannot fail for a row we wrote ourselves.
	e, _ := shared.NewEmail(email)
	s.Email = e
	return &s, nil
}

func (r *StudentRepository) FindByID(ctx context.Context, id string) (*membership.Student, error) {
	query := `SELECT ` + studentColumns + ` FROM students WHERE id = $1`
	s, err := scanStudent(r.db.QueryRow(ctx, query, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, membership.ErrStudentNotFound
	}
	return s, err
}

func (r *StudentRepository) FindByDocumentID(ctx context.Context, documentID string) (*membership.Student, error) {
	query := `SELECT ` + studentColumns + ` FROM students WHERE document_id = $1`
	s, err := scanStudent(r.db.QueryRow(ctx, query, documentID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, membership.ErrStudentNotFound
	}
	return s, err
}

func (r *StudentRepository) Search(ctx context.Context, q string, page, limit int) ([]*membership.Student, int, error) {
	offset := (page - 1) * limit
	query := `SELECT ` + studentColumns + ` FROM students
		WHERE deactivated_at IS NULL AND (full_name ILIKE $1 OR document_id ILIKE $1)
		ORDER BY full_name LIMIT $2 OFFSET $3`

	rows, err := r.db.Query(ctx, query, "%"+q+"%", limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var students []*membership.Student
	for rows.Next() {
		s, err := scanStudent(rows)
		if err != nil {
			return nil, 0, err
		}
		students = append(students, s)
	}

	var total int
	countQuery := `SELECT count(*) FROM students WHERE deactivated_at IS NULL AND (full_name ILIKE $1 OR document_id ILIKE $1)`
	if err := r.db.QueryRow(ctx, countQuery, "%"+q+"%").Scan(&total); err != nil {
		return nil, 0, err
	}

	return students, total, nil
}

func (r *StudentRepository) Save(ctx context.Context, s *membership.Student) error {
	const query = `
		INSERT INTO students (id, full_name, document_id, email, phone, suspended_until, deactivated_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NULLIF($5, ''), $6, $7, $8, $9)
		ON CONFLICT (id) DO UPDATE SET
			full_name = EXCLUDED.full_name,
			email = EXCLUDED.email,
			phone = EXCLUDED.phone,
			suspended_until = EXCLUDED.suspended_until,
			deactivated_at = EXCLUDED.deactivated_at,
			updated_at = EXCLUDED.updated_at`

	_, err := r.db.Exec(ctx, query, s.ID, s.FullName, s.DocumentID, s.Email.String(), s.Phone,
		s.SuspendedUntil, s.DeactivatedAt, s.CreatedAt, s.UpdatedAt)
	return err
}
