package store

import (
	"context"

	"workshop-api/internal/model"
)

// TransitionOrderStatus moves an order to newStatus following the allowed
// lifecycle, appends a status history entry and returns the updated order. An
// illegal transition is a conflict. The status ticket implements the body.
func (s *Store) TransitionOrderStatus(ctx context.Context, orderNumber, newStatus string) (model.Order, error) {
	return model.Order{}, ErrNotImplemented
}
