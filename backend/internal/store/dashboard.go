package store

import (
	"context"

	"workshop-api/internal/model"
)

// dashboardOrderColumns is the projection every dashboard order list shares. It
// joins the vehicle plate and the customer name onto the order and sums the
// line totals of the order's positions; a positionless order reports 0. All
// money stays a whole cent (::bigint), never a float (AC-04, AC-21).
const dashboardOrderColumns = `
	o.order_number,
	o.status,
	v.plate,
	COALESCE(to_char(o.preferred_date, 'YYYY-MM-DD'), '') AS preferred_date,
	c.name,
	COALESCE(SUM(oi.line_total_cents), 0)::bigint AS total_cents`

// dashboardOrderGroupBy lists every non-aggregated column of the projection so
// the SUM groups per order.
const dashboardOrderGroupBy = `
	GROUP BY o.id, o.order_number, o.status, v.plate, o.preferred_date, c.name, o.created_at`

// Dashboard returns the open orders (angefragt, bestaetigt, in_arbeit), the
// orders that changed to 'fertig' today (UTC) and the revenue of the current
// calendar month (UTC) in whole cents (AC-21). Every query is parametrized and
// never logs PII (AC-28, AC-35).
func (s *Store) Dashboard(ctx context.Context) (open []model.OrderRow, finishedToday []model.OrderRow, revenueMonthCents int64, err error) {
	open, err = s.queryDashboardOrders(ctx, `
		SELECT`+dashboardOrderColumns+`
		  FROM workshop_orders o
		  JOIN vehicles v ON v.id = o.vehicle_id
		  JOIN customers c ON c.id = o.customer_id
		  LEFT JOIN order_items oi ON oi.order_id = o.id
		 WHERE o.status = ANY($1)
		`+dashboardOrderGroupBy+`
		 ORDER BY o.created_at, o.id`,
		[]string{model.StatusRequested, model.StatusConfirmed, model.StatusInProgress},
	)
	if err != nil {
		return nil, nil, 0, err
	}

	finishedToday, err = s.queryDashboardOrders(ctx, `
		SELECT`+dashboardOrderColumns+`
		  FROM workshop_orders o
		  JOIN vehicles v ON v.id = o.vehicle_id
		  JOIN customers c ON c.id = o.customer_id
		  LEFT JOIN order_items oi ON oi.order_id = o.id
		 WHERE o.id IN (
		     SELECT h.order_id
		       FROM order_status_history h
		      WHERE h.status = $1
		        AND (h.changed_at AT TIME ZONE 'UTC')::date = (now() AT TIME ZONE 'UTC')::date
		 )
		`+dashboardOrderGroupBy+`
		 ORDER BY o.created_at, o.id`,
		model.StatusDone,
	)
	if err != nil {
		return nil, nil, 0, err
	}

	// The month window is computed in UTC and compared against the timestamptz
	// column, so the monthly revenue follows the calendar month (AC-21). The
	// SUM of a BIGINT stays an integer cent, never a float.
	if err = s.pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(total_cents), 0)::bigint
		  FROM invoices
		 WHERE created_at >= date_trunc('month', now() AT TIME ZONE 'UTC') AT TIME ZONE 'UTC'
		   AND created_at < (date_trunc('month', now() AT TIME ZONE 'UTC') + interval '1 month') AT TIME ZONE 'UTC'`,
	).Scan(&revenueMonthCents); err != nil {
		return nil, nil, 0, err
	}

	return open, finishedToday, revenueMonthCents, nil
}

// queryDashboardOrders runs an order-list projection and scans it into rows. It
// returns an empty slice (never nil) so the JSON response is [] and not null.
func (s *Store) queryDashboardOrders(ctx context.Context, sql string, args ...any) ([]model.OrderRow, error) {
	rows, err := s.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []model.OrderRow{}
	for rows.Next() {
		var row model.OrderRow
		if err := rows.Scan(
			&row.OrderNumber,
			&row.Status,
			&row.Plate,
			&row.PreferredDate,
			&row.CustomerName,
			&row.TotalCents,
		); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
