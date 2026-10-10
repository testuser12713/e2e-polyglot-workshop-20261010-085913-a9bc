package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"workshop-api/internal/model"
)

// ErrOrderLookupNotFound is returned by the public order and invoice reads for
// every mismatch — an unknown order number, a wrong plate or any other
// combination that does not identify the order. The HTTP layer answers all of
// them with one neutral 404 that never reveals whether the order exists
// (AC-12).
var ErrOrderLookupNotFound = errors.New("order lookup not found")

// OrderByNumberAndPlate returns the order matching the order number and the
// normalized plate together with its status history. A mismatch is a not-found
// result. Both keys are compared as parameters, never concatenated into SQL
// (AC-28).
func (s *Store) OrderByNumberAndPlate(ctx context.Context, orderNumber, plateNormalized string) (model.Order, []model.StatusEvent, error) {
	var order model.Order
	err := s.pool.QueryRow(ctx, `
		SELECT o.id, o.order_number, o.customer_id, o.vehicle_id, o.status,
		       o.preferred_date, o.description, o.created_at, o.updated_at
		  FROM workshop_orders o
		  JOIN vehicles v ON v.id = o.vehicle_id
		 WHERE o.order_number = $1
		   AND v.plate_normalized = $2`,
		orderNumber, plateNormalized).
		Scan(&order.ID, &order.OrderNumber, &order.CustomerID, &order.VehicleID,
			&order.Status, &order.PreferredDate, &order.Description,
			&order.CreatedAt, &order.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Order{}, nil, ErrOrderLookupNotFound
	}
	if err != nil {
		return model.Order{}, nil, err
	}
	order.CreatedAt = order.CreatedAt.UTC()
	order.UpdatedAt = order.UpdatedAt.UTC()
	order.PreferredDate = order.PreferredDate.UTC()

	history, err := statusHistory(ctx, s.pool, order.ID)
	if err != nil {
		return model.Order{}, nil, err
	}
	return order, history, nil
}

// InvoiceByOrderNumberAndPlate returns the invoice for a finished order that
// matches the order number and normalized plate. Every mismatch — unknown order
// number, wrong plate, an order that has not reached 'fertig' yet or a missing
// invoice — is reported as ErrOrderLookupNotFound so the HTTP layer can answer
// one neutral response (AC-13). The invoice stays available once the order has
// moved on to 'abgeholt', the normal end of the lifecycle.
func (s *Store) InvoiceByOrderNumberAndPlate(ctx context.Context, orderNumber, plateNormalized string) (model.Invoice, error) {
	var invoiceID int64
	err := s.pool.QueryRow(ctx, `
		SELECT i.id
		  FROM invoices i
		  JOIN workshop_orders o ON o.order_number = i.order_number
		  JOIN vehicles v ON v.id = o.vehicle_id
		 WHERE i.order_number = $1
		   AND v.plate_normalized = $2
		   AND o.status = ANY($3)`,
		orderNumber, plateNormalized, []string{model.StatusDone, model.StatusPickedUp}).Scan(&invoiceID)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Invoice{}, ErrOrderLookupNotFound
	}
	if err != nil {
		return model.Invoice{}, err
	}
	return s.invoiceByID(ctx, invoiceID)
}
