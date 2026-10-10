package store

import (
	"context"

	"workshop-api/internal/model"
)

// UpsertCustomerByEmail returns the customer with the given email, creating it
// when it does not exist yet. A repeat email reuses the existing row (the
// customers_email_uidx unique index decides this in the database, not in
// application code) and refreshes its name and phone (AC-01). The query is
// parametrized (AC-28); no field of the customer is ever logged (AC-35).
func (s *Store) UpsertCustomerByEmail(ctx context.Context, name, email, phone string) (model.Customer, error) {
	const query = `
		INSERT INTO customers (name, email, phone)
		VALUES ($1, $2, $3)
		ON CONFLICT (email) DO UPDATE
		SET name = EXCLUDED.name, phone = EXCLUDED.phone
		RETURNING id, name, email, phone, created_at`

	var c model.Customer
	err := s.pool.QueryRow(ctx, query, name, email, phone).
		Scan(&c.ID, &c.Name, &c.Email, &c.Phone, &c.CreatedAt)
	if err != nil {
		return model.Customer{}, err
	}
	return c, nil
}
