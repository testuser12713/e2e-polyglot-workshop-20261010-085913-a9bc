package store

import (
	"context"

	"workshop-api/internal/model"
)

// ListOrders returns order rows filtered by status (exact) and by plate
// (case-insensitive substring, both optional). The order list ticket implements
// the body.
func (s *Store) ListOrders(ctx context.Context, status, plateQuery string) ([]model.OrderRow, error) {
	return nil, ErrNotImplemented
}
