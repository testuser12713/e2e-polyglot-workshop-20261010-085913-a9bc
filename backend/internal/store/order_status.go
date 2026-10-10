package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"workshop-api/internal/model"
)

// Sentinel errors of the status transition. The HTTP layer maps them to the
// stable error codes of the API.
var (
	// ErrOrderNotFound is returned when no order carries the given number.
	ErrOrderNotFound = errors.New("order not found")
	// ErrStatusConflict is returned for every transition that the order
	// lifecycle does not allow. The order is left untouched.
	ErrStatusConflict = errors.New("status transition not allowed")
	// ErrPublishFailed is returned when the invoice message for a finished
	// order could not be published. The whole transition is rolled back.
	ErrPublishFailed = errors.New("could not publish invoice message")
)

// StatusPublisher publishes the invoice message of a finished order. It is
// invoked inside the transition transaction so that a failure rolls the status
// change back instead of silently succeeding.
type StatusPublisher func(ctx context.Context, orderNumber, plate string) error

// allowedTransitions is the only lifecycle the workshop allows:
// angefragt -> bestaetigt -> in_arbeit -> fertig -> abgeholt (AC-05).
var allowedTransitions = map[string]string{
	model.StatusRequested:  model.StatusConfirmed,
	model.StatusConfirmed:  model.StatusInProgress,
	model.StatusInProgress: model.StatusDone,
	model.StatusDone:       model.StatusPickedUp,
}

// TransitionOrderStatus moves an order to newStatus following the allowed
// lifecycle, appends a status history entry and returns the updated order with
// its full history. An illegal transition is a conflict; when moving to
// 'fertig' the publish callback runs inside the transaction so that a publish
// failure (AC-07) rolls the change back.
func (s *Store) TransitionOrderStatus(ctx context.Context, orderNumber, newStatus string, publish StatusPublisher) (model.Order, []model.StatusEvent, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return model.Order{}, nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var (
		orderID int64
		current string
		plate   string
	)
	err = tx.QueryRow(ctx, `
		SELECT o.id, o.status, v.plate
		  FROM workshop_orders o
		  JOIN vehicles v ON v.id = o.vehicle_id
		 WHERE o.order_number = $1
		 FOR UPDATE OF o`, orderNumber).Scan(&orderID, &current, &plate)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Order{}, nil, ErrOrderNotFound
	}
	if err != nil {
		return model.Order{}, nil, err
	}

	if expected, ok := allowedTransitions[current]; !ok || expected != newStatus {
		return model.Order{}, nil, ErrStatusConflict
	}

	changedAt := time.Now().UTC()
	if _, err := tx.Exec(ctx, `
		UPDATE workshop_orders
		   SET status = $1, updated_at = $2
		 WHERE id = $3`, newStatus, changedAt, orderID); err != nil {
		return model.Order{}, nil, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO order_status_history (order_id, status, changed_at)
		VALUES ($1, $2, $3)`, orderID, newStatus, changedAt); err != nil {
		return model.Order{}, nil, err
	}

	if newStatus == model.StatusDone {
		if publish == nil {
			return model.Order{}, nil, ErrPublishFailed
		}
		if err := publish(ctx, orderNumber, plate); err != nil {
			return model.Order{}, nil, ErrPublishFailed
		}
	}

	history, err := statusHistory(ctx, tx, orderID)
	if err != nil {
		return model.Order{}, nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return model.Order{}, nil, err
	}

	return model.Order{ID: orderID, OrderNumber: orderNumber, Status: newStatus, UpdatedAt: changedAt}, history, nil
}

// statusHistory reads the append-only history of an order in chronological
// order. It runs on the caller's queryer (a transaction or the pool) so it sees
// the row just appended when called inside a transition.
func statusHistory(ctx context.Context, q queryer, orderID int64) ([]model.StatusEvent, error) {
	rows, err := q.Query(ctx, `
		SELECT status, changed_at
		  FROM order_status_history
		 WHERE order_id = $1
		 ORDER BY changed_at, id`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := []model.StatusEvent{}
	for rows.Next() {
		var e model.StatusEvent
		if err := rows.Scan(&e.Status, &e.ChangedAt); err != nil {
			return nil, err
		}
		e.ChangedAt = e.ChangedAt.UTC()
		events = append(events, e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return events, nil
}
