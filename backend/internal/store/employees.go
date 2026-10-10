package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"

	"workshop-api/internal/model"
)

// ErrEmployeeNotFound is returned by the employee lookup when no employee
// carries the requested e-mail. The login route turns it into the same 401 as a
// wrong password so the two are indistinguishable (AC-29).
var ErrEmployeeNotFound = errors.New("employee_not_found")

// EnsureFirstEmployee creates the configured workshop employee if it does not
// exist yet. It is idempotent: a second startup neither creates a second row
// nor overwrites an existing password (AC-15).
func (s *Store) EnsureFirstEmployee(ctx context.Context, email, password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx,
		`INSERT INTO employees (email, password_hash) VALUES ($1, $2)
		 ON CONFLICT (email) DO NOTHING`,
		email, string(hash),
	)
	return err
}

// FindEmployeeByEmail returns the employee with the given e-mail together with
// its bcrypt password hash. It returns ErrEmployeeNotFound when no such
// employee exists.
func (s *Store) FindEmployeeByEmail(ctx context.Context, email string) (model.Employee, string, error) {
	var (
		emp  model.Employee
		hash string
	)
	err := s.pool.QueryRow(ctx,
		`SELECT id, email, created_at, password_hash FROM employees WHERE email = $1`,
		email,
	).Scan(&emp.ID, &emp.Email, &emp.CreatedAt, &hash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Employee{}, "", ErrEmployeeNotFound
		}
		return model.Employee{}, "", err
	}
	return emp, hash, nil
}

// EmployeeByEmail is the skeleton's declared seam; it delegates to
// FindEmployeeByEmail to keep that name working.
func (s *Store) EmployeeByEmail(ctx context.Context, email string) (model.Employee, string, error) {
	return s.FindEmployeeByEmail(ctx, email)
}
