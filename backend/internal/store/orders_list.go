package store

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"

	"workshop-api/internal/model"
)

// ListOrders returns order rows filtered by status (exact) and by plate
// (case-insensitive substring, both optional). Rows carry the summed position
// total and the customer name and are ordered by preferred date (nulls last)
// then order number.
//
// Both filter values are passed as bound query parameters. The plate pattern is
// assembled inside SQL from the bound parameter, so the request value is never
// concatenated into the SQL text (AC-28).
func (s *Store) ListOrders(ctx context.Context, status, plateQuery string) ([]model.OrderRow, error) {
	const query = `
SELECT o.order_number,
       o.status,
       v.plate,
       o.preferred_date,
       c.name,
       COALESCE(SUM(oi.line_total_cents), 0) AS total_cents
FROM workshop_orders o
JOIN customers c ON c.id = o.customer_id
JOIN vehicles v ON v.id = o.vehicle_id
LEFT JOIN order_items oi ON oi.order_id = o.id
WHERE ($1 = '' OR o.status = $1)
  AND ($2 = '' OR v.plate ILIKE '%' || $2 || '%')
GROUP BY o.id, o.order_number, o.status, v.plate, o.preferred_date, c.name
ORDER BY o.preferred_date ASC NULLS LAST, o.order_number ASC`

	rows, err := s.pool.Query(ctx, query, status, plateQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	orders := make([]model.OrderRow, 0)
	for rows.Next() {
		var (
			row       model.OrderRow
			preferred pgtype.Date
		)
		if err := rows.Scan(&row.OrderNumber, &row.Status, &row.Plate, &preferred, &row.CustomerName, &row.TotalCents); err != nil {
			return nil, err
		}
		if preferred.Valid {
			row.PreferredDate = preferred.Time.Format("2006-01-02")
		}
		orders = append(orders, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return orders, nil
}
