package store

import (
	"context"

	"workshop-api/internal/model"
)

// AddOrderItem inserts a position into an order and returns it with its line
// total. The order item ticket implements the body.
func (s *Store) AddOrderItem(ctx context.Context, orderNumber string, item model.OrderItem) (model.OrderItem, error) {
	return model.OrderItem{}, ErrNotImplemented
}

// UpdateOrderItem updates an existing position of an order and returns it.
func (s *Store) UpdateOrderItem(ctx context.Context, orderNumber string, itemID int64, item model.OrderItem) (model.OrderItem, error) {
	return model.OrderItem{}, ErrNotImplemented
}

// DeleteOrderItem removes a position from an order.
func (s *Store) DeleteOrderItem(ctx context.Context, orderNumber string, itemID int64) error {
	return ErrNotImplemented
}
