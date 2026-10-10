package store

import (
	"context"

	"golang.org/x/crypto/bcrypt"

	"workshop-api/internal/model"
)

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

// EmployeeByEmail returns the employee with the given email and its password
// hash. The login ticket implements the body.
func (s *Store) EmployeeByEmail(ctx context.Context, email string) (model.Employee, string, error) {
	return model.Employee{}, "", ErrNotImplemented
}
