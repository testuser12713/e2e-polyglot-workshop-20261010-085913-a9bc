package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"workshop-api/internal/model"
)

// InvoiceByOrderNumber returns the invoice of an order with its positions. An
// order without a stored invoice is reported as ErrOrderLookupNotFound.
func (s *Store) InvoiceByOrderNumber(ctx context.Context, orderNumber string) (model.Invoice, error) {
	var invoiceID int64
	err := s.pool.QueryRow(ctx,
		`SELECT id FROM invoices WHERE order_number = $1`, orderNumber).Scan(&invoiceID)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Invoice{}, ErrOrderLookupNotFound
	}
	if err != nil {
		return model.Invoice{}, err
	}
	return s.invoiceByID(ctx, invoiceID)
}

// invoiceByID loads one invoice header and its positions. All amounts are whole
// cents stored in the database; the missing numeric columns of a position are
// read as zero so the domain type stays non-nullable.
func (s *Store) invoiceByID(ctx context.Context, invoiceID int64) (model.Invoice, error) {
	var invoice model.Invoice
	err := s.pool.QueryRow(ctx, `
		SELECT id, order_number, subtotal_cents, tax_cents, total_cents, created_at
		  FROM invoices
		 WHERE id = $1`, invoiceID).
		Scan(&invoice.ID, &invoice.OrderNumber, &invoice.SubtotalCents,
			&invoice.TaxCents, &invoice.TotalCents, &invoice.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Invoice{}, ErrOrderLookupNotFound
	}
	if err != nil {
		return model.Invoice{}, err
	}
	invoice.CreatedAt = invoice.CreatedAt.UTC()

	items, err := invoiceItems(ctx, s.pool, invoiceID)
	if err != nil {
		return model.Invoice{}, err
	}
	invoice.Items = items
	return invoice, nil
}

// invoiceItems reads the positions of an invoice in insertion order.
func invoiceItems(ctx context.Context, q queryer, invoiceID int64) ([]model.InvoiceItem, error) {
	rows, err := q.Query(ctx, `
		SELECT kind, description,
		       COALESCE(hours, 0)::float8,
		       COALESCE(quantity, 0),
		       COALESCE(unit_price_cents, 0),
		       line_total_cents
		  FROM invoice_items
		 WHERE invoice_id = $1
		 ORDER BY id`, invoiceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []model.InvoiceItem{}
	for rows.Next() {
		var item model.InvoiceItem
		if err := rows.Scan(&item.Kind, &item.Description, &item.Hours,
			&item.Quantity, &item.UnitPriceCents, &item.LineTotalCents); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
