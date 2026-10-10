package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"workshop-api/internal/model"
)

// OrderDetail returns the full order (customer, vehicle, positions, status
// history and subtotal) for the workshop detail view. An unknown order number
// yields ErrOrderNotFound (declared with the status-transition sentinels in
// order_status.go).
func (s *Store) OrderDetail(ctx context.Context, orderNumber string) (model.OrderDetail, error) {
	var (
		detail    model.OrderDetail
		preferred pgtype.Date
	)

	err := s.pool.QueryRow(ctx, `
SELECT o.order_number,
       o.status,
       v.plate,
       o.preferred_date,
       c.name,
       COALESCE((SELECT SUM(oi.line_total_cents) FROM order_items oi WHERE oi.order_id = o.id), 0) AS subtotal_cents,
       o.description,
       c.id, c.email, c.phone, c.created_at,
       v.id, v.plate_normalized, v.brand, v.model, v.mileage, v.created_at
FROM workshop_orders o
JOIN customers c ON c.id = o.customer_id
JOIN vehicles v ON v.id = o.vehicle_id
WHERE o.order_number = $1`, orderNumber).Scan(
		&detail.OrderNumber,
		&detail.Status,
		&detail.Plate,
		&preferred,
		&detail.CustomerName,
		&detail.SubtotalCents,
		&detail.Description,
		&detail.Customer.ID,
		&detail.Customer.Email,
		&detail.Customer.Phone,
		&detail.Customer.CreatedAt,
		&detail.Vehicle.ID,
		&detail.Vehicle.PlateNormalized,
		&detail.Vehicle.Brand,
		&detail.Vehicle.Model,
		&detail.Vehicle.Mileage,
		&detail.Vehicle.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.OrderDetail{}, ErrOrderNotFound
	}
	if err != nil {
		return model.OrderDetail{}, err
	}
	if preferred.Valid {
		detail.PreferredDate = preferred.Time.Format("2006-01-02")
	}

	detail.TotalCents = detail.SubtotalCents
	detail.Customer.Name = detail.CustomerName
	detail.Vehicle.Plate = detail.Plate

	items, err := s.orderItemsForDetail(ctx, orderNumber)
	if err != nil {
		return model.OrderDetail{}, err
	}
	detail.Items = items

	history, err := s.orderStatusHistoryForDetail(ctx, orderNumber)
	if err != nil {
		return model.OrderDetail{}, err
	}
	detail.StatusHistory = history

	return detail, nil
}

// orderItemsForDetail loads the positions of an order in insertion order.
func (s *Store) orderItemsForDetail(ctx context.Context, orderNumber string) ([]model.OrderItem, error) {
	rows, err := s.pool.Query(ctx, `
SELECT oi.id,
       oi.order_id,
       oi.kind,
       oi.description,
       COALESCE(oi.hours, 0),
       COALESCE(oi.quantity, 0),
       COALESCE(oi.unit_price_cents, 0),
       oi.line_total_cents
FROM order_items oi
JOIN workshop_orders o ON o.id = oi.order_id
WHERE o.order_number = $1
ORDER BY oi.id`, orderNumber)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.OrderItem, 0)
	for rows.Next() {
		var item model.OrderItem
		if err := rows.Scan(
			&item.ID,
			&item.OrderID,
			&item.Kind,
			&item.Description,
			&item.Hours,
			&item.Quantity,
			&item.UnitPriceCents,
			&item.LineTotalCents,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

// orderStatusHistoryForDetail loads the status history of an order oldest first.
func (s *Store) orderStatusHistoryForDetail(ctx context.Context, orderNumber string) ([]model.StatusEvent, error) {
	rows, err := s.pool.Query(ctx, `
SELECT osh.status, osh.changed_at
FROM order_status_history osh
JOIN workshop_orders o ON o.id = osh.order_id
WHERE o.order_number = $1
ORDER BY osh.changed_at ASC, osh.id ASC`, orderNumber)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	history := make([]model.StatusEvent, 0)
	for rows.Next() {
		var event model.StatusEvent
		if err := rows.Scan(&event.Status, &event.ChangedAt); err != nil {
			return nil, err
		}
		history = append(history, event)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return history, nil
}
