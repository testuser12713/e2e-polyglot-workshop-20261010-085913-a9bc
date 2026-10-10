package store

import (
	"context"

	"workshop-api/internal/model"
)

// OrderDetail returns the full order (customer, vehicle, positions, status
// history and subtotal) for the workshop detail view. The order list ticket
// implements the body.
func (s *Store) OrderDetail(ctx context.Context, orderNumber string) (model.OrderDetail, error) {
	return model.OrderDetail{}, ErrNotImplemented
}
