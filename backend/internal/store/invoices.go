package store

import (
	"context"

	"workshop-api/internal/model"
)

// InvoiceByOrderNumber returns the invoice of an order with its positions.
func (s *Store) InvoiceByOrderNumber(ctx context.Context, orderNumber string) (model.Invoice, error) {
	return model.Invoice{}, ErrNotImplemented
}
