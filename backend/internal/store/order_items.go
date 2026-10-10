package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"workshop-api/internal/model"
)

// Sentinel errors of the order item store. The HTTP layer maps them to the
// shared error envelope: unknown order or item -> 404, a collected order -> 409.
var (
	// ErrOrderNotFound means no workshop order carries the given order number.
	ErrOrderNotFound = errors.New("order_not_found")
	// ErrItemNotFound means the order exists but has no position with that id.
	ErrItemNotFound = errors.New("item_not_found")
	// ErrOrderLocked means the order is in status "abgeholt" and is read-only.
	ErrOrderLocked = errors.New("order_locked")
)

// orderIDAndStatus resolves an order number to its primary key and status. It
// is the single lookup every item mutation uses so the 404/409 rules stay in
// one place.
func (s *Store) orderIDAndStatus(ctx context.Context, orderNumber string) (int64, string, error) {
	var (
		id     int64
		status string
	)
	err := s.pool.QueryRow(ctx,
		`SELECT id, status FROM workshop_orders WHERE order_number = $1`,
		orderNumber,
	).Scan(&id, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, "", ErrOrderNotFound
	}
	if err != nil {
		return 0, "", err
	}
	return id, status, nil
}

// scanOrderItem reads one order_items row. NULL hours, quantity and unit price
// collapse to their zero value so the JSON response stays readable.
func scanOrderItem(row pgx.Row) (model.OrderItem, error) {
	var item model.OrderItem
	err := row.Scan(
		&item.ID,
		&item.OrderID,
		&item.Kind,
		&item.Description,
		&item.Hours,
		&item.Quantity,
		&item.UnitPriceCents,
		&item.LineTotalCents,
	)
	return item, err
}

const orderItemColumns = `id, order_id, kind, description,
	COALESCE(hours, 0), COALESCE(quantity, 0),
	COALESCE(unit_price_cents, 0), line_total_cents`

// AddOrderItem inserts a position into an order and returns it with its line
// total. The caller has already validated the position; this method owns the
// existence and collected-status rules (AC-04, AC-19).
func (s *Store) AddOrderItem(ctx context.Context, orderNumber string, item model.OrderItem) (model.OrderItem, error) {
	orderID, status, err := s.orderIDAndStatus(ctx, orderNumber)
	if err != nil {
		return model.OrderItem{}, err
	}
	if status == model.StatusPickedUp {
		return model.OrderItem{}, ErrOrderLocked
	}

	var (
		hoursArg any
		qtyArg   any
	)
	if item.Kind == model.ItemKindLabour {
		hoursArg = item.Hours
	}
	if item.Kind == model.ItemKindPart {
		qtyArg = item.Quantity
	}

	row := s.pool.QueryRow(ctx,
		`INSERT INTO order_items
			(order_id, kind, description, hours, quantity, unit_price_cents, line_total_cents)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING `+orderItemColumns,
		orderID, item.Kind, item.Description, hoursArg, qtyArg, item.UnitPriceCents, item.LineTotalCents,
	)
	created, err := scanOrderItem(row)
	if err != nil {
		return model.OrderItem{}, err
	}
	return created, nil
}

// OrderItemByID returns a single position of an order. It lets the PATCH
// handler merge a partial request onto the stored position before validating
// the result.
func (s *Store) OrderItemByID(ctx context.Context, orderNumber string, itemID int64) (model.OrderItem, error) {
	orderID, _, err := s.orderIDAndStatus(ctx, orderNumber)
	if err != nil {
		return model.OrderItem{}, err
	}
	row := s.pool.QueryRow(ctx,
		`SELECT `+orderItemColumns+` FROM order_items WHERE id = $1 AND order_id = $2`,
		itemID, orderID,
	)
	item, err := scanOrderItem(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.OrderItem{}, ErrItemNotFound
	}
	if err != nil {
		return model.OrderItem{}, err
	}
	return item, nil
}

// UpdateOrderItem updates an existing position of an order and returns it. A
// collected order refuses every change (AC-05, AC-19).
func (s *Store) UpdateOrderItem(ctx context.Context, orderNumber string, itemID int64, item model.OrderItem) (model.OrderItem, error) {
	orderID, status, err := s.orderIDAndStatus(ctx, orderNumber)
	if err != nil {
		return model.OrderItem{}, err
	}
	if status == model.StatusPickedUp {
		return model.OrderItem{}, ErrOrderLocked
	}

	var (
		hoursArg any
		qtyArg   any
	)
	if item.Kind == model.ItemKindLabour {
		hoursArg = item.Hours
	}
	if item.Kind == model.ItemKindPart {
		qtyArg = item.Quantity
	}

	row := s.pool.QueryRow(ctx,
		`UPDATE order_items
		    SET kind = $1, description = $2, hours = $3, quantity = $4,
		        unit_price_cents = $5, line_total_cents = $6
		  WHERE id = $7 AND order_id = $8
		  RETURNING `+orderItemColumns,
		item.Kind, item.Description, hoursArg, qtyArg, item.UnitPriceCents, item.LineTotalCents,
		itemID, orderID,
	)
	updated, err := scanOrderItem(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.OrderItem{}, ErrItemNotFound
	}
	if err != nil {
		return model.OrderItem{}, err
	}
	return updated, nil
}

// DeleteOrderItem removes a position from an order. A collected order refuses
// the delete (AC-05, AC-19).
func (s *Store) DeleteOrderItem(ctx context.Context, orderNumber string, itemID int64) error {
	orderID, status, err := s.orderIDAndStatus(ctx, orderNumber)
	if err != nil {
		return err
	}
	if status == model.StatusPickedUp {
		return ErrOrderLocked
	}

	tag, err := s.pool.Exec(ctx,
		`DELETE FROM order_items WHERE id = $1 AND order_id = $2`,
		itemID, orderID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrItemNotFound
	}
	return nil
}
