package store

import (
	"context"

	"workshop-api/internal/model"
)

// Dashboard returns the open orders, the orders finished today and the revenue
// of the current month in whole cents. The dashboard ticket implements the body.
func (s *Store) Dashboard(ctx context.Context) (open []model.OrderRow, finishedToday []model.OrderRow, revenueMonthCents int64, err error) {
	return nil, nil, 0, ErrNotImplemented
}
