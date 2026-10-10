package store

import (
	"context"

	"workshop-api/internal/model"
)

// UpsertCustomer returns the customer with the given email, creating it when it
// does not exist yet. The public appointment ticket implements the body.
func (s *Store) UpsertCustomer(ctx context.Context, name, email, phone string) (model.Customer, error) {
	return model.Customer{}, ErrNotImplemented
}
