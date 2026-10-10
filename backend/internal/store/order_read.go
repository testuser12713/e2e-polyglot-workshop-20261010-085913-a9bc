package store

import (
	"context"

	"workshop-api/internal/model"
)

// OrderByNumberAndPlate returns the order matching the order number and the
// normalized plate together with its status history. A mismatch is a not-found
// result. The public read ticket implements the body.
func (s *Store) OrderByNumberAndPlate(ctx context.Context, orderNumber, plateNormalized string) (model.Order, []model.StatusEvent, error) {
	return model.Order{}, nil, ErrNotImplemented
}

// InvoiceByOrderNumberAndPlate returns the invoice for a finished order that
// matches the order number and normalized plate. The public read ticket
// implements the body.
func (s *Store) InvoiceByOrderNumberAndPlate(ctx context.Context, orderNumber, plateNormalized string) (model.Invoice, error) {
	return model.Invoice{}, ErrNotImplemented
}
