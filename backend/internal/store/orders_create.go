package store

import (
	"context"
	"time"

	"workshop-api/internal/model"
)

// CreateOrder inserts a new order in status "angefragt" and returns it with its
// generated order number (AC-03). The order number is derived from the current
// year and the table's own identity sequence, so it is unique without an extra
// round trip; the unique index additionally guards it. The first status history
// entry is written in the same transaction with changed_at in UTC (AC-06). All
// queries are parametrized (AC-28); the description is never logged (AC-35).
func (s *Store) CreateOrder(ctx context.Context, customerID, vehicleID int64, preferredDate time.Time, description string) (model.Order, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return model.Order{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	const insertOrder = `
		INSERT INTO workshop_orders (order_number, customer_id, vehicle_id, status, preferred_date, description)
		VALUES (
			'AUF-' || to_char($3::date, 'YYYY') || '-' ||
				lpad(nextval(pg_get_serial_sequence('workshop_orders', 'id'))::text, 4, '0'),
			$1, $2, $4, $3, $5
		)
		RETURNING id, order_number, customer_id, vehicle_id, status, preferred_date, description, created_at, updated_at`

	var o model.Order
	err = tx.QueryRow(ctx, insertOrder, customerID, vehicleID, preferredDate, model.StatusRequested, description).
		Scan(&o.ID, &o.OrderNumber, &o.CustomerID, &o.VehicleID, &o.Status, &o.PreferredDate, &o.Description, &o.CreatedAt, &o.UpdatedAt)
	if err != nil {
		return model.Order{}, err
	}

	const insertHistory = `
		INSERT INTO order_status_history (order_id, status, changed_at)
		VALUES ($1, $2, $3)`
	if _, err = tx.Exec(ctx, insertHistory, o.ID, model.StatusRequested, time.Now().UTC()); err != nil {
		return model.Order{}, err
	}

	if err = tx.Commit(ctx); err != nil {
		return model.Order{}, err
	}
	return o, nil
}
