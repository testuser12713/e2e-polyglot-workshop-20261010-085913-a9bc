package store

import (
	"context"
	"time"

	"workshop-api/internal/model"
)

// CreateOrder inserts a new order in status "angefragt" and returns it with its
// generated order number (AC-03). The public appointment ticket implements the body.
func (s *Store) CreateOrder(ctx context.Context, customerID, vehicleID int64, preferredDate time.Time, description string) (model.Order, error) {
	return model.Order{}, ErrNotImplemented
}
