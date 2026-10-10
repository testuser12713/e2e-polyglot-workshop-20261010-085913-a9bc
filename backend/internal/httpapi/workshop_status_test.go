package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	valkey "github.com/valkey-io/valkey-go"

	"workshop-api/internal/model"
	"workshop-api/internal/queue"
	"workshop-api/internal/store"
)

// statusEnv is one isolated order seeded at 'angefragt' plus the real
// PostgreSQL store, the real Valkey publisher and a raw client to inspect the
// invoice stream (AC-27).
type statusEnv struct {
	store  *store.Store
	pub    *queue.Publisher
	client valkey.Client
	order  string
	plate  string
}

type statusResp struct {
	OrderNumber   string              `json:"order_number"`
	Status        string              `json:"status"`
	StatusHistory []model.StatusEvent `json:"status_history"`
}

// newStatusEnv connects to the real PostgreSQL and Valkey declared by
// DATABASE_URL and VALKEY_URL, applies the product migration and seeds one
// unique order in status 'angefragt' with an initial history entry.
func newStatusEnv(t *testing.T) statusEnv {
	t.Helper()

	dbURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	valkeyURL := strings.TrimSpace(os.Getenv("VALKEY_URL"))
	if dbURL == "" || valkeyURL == "" {
		t.Skip("DATABASE_URL and VALKEY_URL are required; this test runs against real PostgreSQL and Valkey")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("connect to PostgreSQL: %v", err)
	}
	t.Cleanup(pool.Close)
	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("ping PostgreSQL: %v", err)
	}
	applyTestMigration(t, ctx, pool)

	pub, err := queue.NewPublisher(valkeyURL)
	if err != nil {
		t.Fatalf("connect to Valkey: %v", err)
	}
	t.Cleanup(pub.Close)

	opt, err := valkey.ParseURL(valkeyURL)
	if err != nil {
		t.Fatalf("parse Valkey URL: %v", err)
	}
	client, err := valkey.NewClient(opt)
	if err != nil {
		t.Fatalf("open Valkey reader: %v", err)
	}
	t.Cleanup(client.Close)

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	orderNumber := "SW-" + suffix
	plate := "B-STATUS-" + suffix
	email := "status-" + suffix + "@example.test"

	var customerID, vehicleID, orderID int64
	if err := pool.QueryRow(ctx,
		`INSERT INTO customers (name, email, phone) VALUES ($1, $2, $3) RETURNING id`,
		"Status Tester", email, "").Scan(&customerID); err != nil {
		t.Fatalf("insert customer: %v", err)
	}
	if err := pool.QueryRow(ctx,
		`INSERT INTO vehicles (plate, plate_normalized, brand, model, mileage)
		 VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		plate, strings.ToUpper(plate), "VW", "Golf", 1000).Scan(&vehicleID); err != nil {
		t.Fatalf("insert vehicle: %v", err)
	}
	if err := pool.QueryRow(ctx,
		`INSERT INTO workshop_orders (order_number, customer_id, vehicle_id, status, preferred_date, description)
		 VALUES ($1, $2, $3, $4, now()::date, '') RETURNING id`,
		orderNumber, customerID, vehicleID, model.StatusRequested).Scan(&orderID); err != nil {
		t.Fatalf("insert order: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO order_status_history (order_id, status, changed_at) VALUES ($1, $2, $3)`,
		orderID, model.StatusRequested, time.Now().UTC().Add(-time.Hour)); err != nil {
		t.Fatalf("insert initial history: %v", err)
	}

	return statusEnv{
		store:  store.New(pool),
		pub:    pub,
		client: client,
		order:  orderNumber,
		plate:  plate,
	}
}

// postStatus calls the handler directly with the employee placed in the
// request context via WithEmployee, exactly as the auth middleware will.
func (e statusEnv) postStatus(t *testing.T, status string) *httptest.ResponseRecorder {
	t.Helper()
	body := strings.NewReader(fmt.Sprintf(`{"status":%q}`, status))
	req := httptest.NewRequest(http.MethodPost, "/api/workshop/orders/"+e.order+"/status", body)
	req.SetPathValue("order_number", e.order)
	req = req.WithContext(WithEmployee(req.Context(), model.Employee{ID: 1, Email: "chef@werkstatt.example"}))
	rec := httptest.NewRecorder()
	handleSetStatus(e.store, e.pub).ServeHTTP(rec, req)
	return rec
}

func (e statusEnv) currentStatus(t *testing.T) string {
	t.Helper()
	var status string
	if err := e.store.Pool().QueryRow(context.Background(),
		`SELECT status FROM workshop_orders WHERE order_number = $1`, e.order).Scan(&status); err != nil {
		t.Fatalf("read order status: %v", err)
	}
	return status
}

func (e statusEnv) historyCount(t *testing.T) int {
	t.Helper()
	var count int
	if err := e.store.Pool().QueryRow(context.Background(),
		`SELECT count(*) FROM order_status_history h
		   JOIN workshop_orders o ON o.id = h.order_id
		  WHERE o.order_number = $1`, e.order).Scan(&count); err != nil {
		t.Fatalf("count history: %v", err)
	}
	return count
}

// orderMessages returns the invoice stream messages belonging to this test's
// order, so a sibling run sharing the stream does not distort the count.
func (e statusEnv) orderMessages(t *testing.T) []valkey.XRangeEntry {
	t.Helper()
	res := e.client.Do(context.Background(),
		e.client.B().Xrange().Key(queue.InvoiceStream).Start("-").End("+").Build())
	entries, err := res.AsXRange()
	if err != nil {
		t.Fatalf("read invoice stream: %v", err)
	}
	mine := make([]valkey.XRangeEntry, 0, len(entries))
	for _, entry := range entries {
		if entry.FieldValues["order_number"] == e.order {
			mine = append(mine, entry)
		}
	}
	return mine
}

func decodeStatus(t *testing.T, rec *httptest.ResponseRecorder) statusResp {
	t.Helper()
	var resp statusResp
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response %q: %v", rec.Body.String(), err)
	}
	return resp
}

// TestSetStatusAppliesAllowedTransitionsWithUTCHistory walks all four allowed
// transitions and checks that every change is appended to the history with a
// UTC timestamp while the earlier entries stay intact (AC-05, AC-06).
func TestSetStatusAppliesAllowedTransitionsWithUTCHistory(t *testing.T) {
	env := newStatusEnv(t)

	steps := []string{
		model.StatusConfirmed,
		model.StatusInProgress,
		model.StatusDone,
		model.StatusPickedUp,
	}
	for i, want := range steps {
		rec := env.postStatus(t, want)
		if rec.Code != http.StatusOK {
			t.Fatalf("transition %d to %q = %d, want 200 (body %q)", i, want, rec.Code, rec.Body.String())
		}
		resp := decodeStatus(t, rec)
		if resp.OrderNumber != env.order {
			t.Fatalf("order_number = %q, want %q", resp.OrderNumber, env.order)
		}
		if resp.Status != want {
			t.Fatalf("status = %q, want %q", resp.Status, want)
		}
		if len(resp.StatusHistory) != i+2 {
			t.Fatalf("history length after transition %d = %d, want %d", i, len(resp.StatusHistory), i+2)
		}
		if first := resp.StatusHistory[0]; first.Status != model.StatusRequested {
			t.Fatalf("initial history entry lost: first status = %q", first.Status)
		}
		last := resp.StatusHistory[len(resp.StatusHistory)-1]
		if last.Status != want {
			t.Fatalf("last history status = %q, want %q", last.Status, want)
		}
		if loc := last.ChangedAt.Location(); loc != time.UTC {
			t.Fatalf("changed_at location = %v, want UTC", loc)
		}
	}

	if got := env.currentStatus(t); got != model.StatusPickedUp {
		t.Fatalf("final order status = %q, want %q", got, model.StatusPickedUp)
	}
}

// TestSetStatusRejectsForbiddenTransitionWithoutChange asserts that an illegal
// transition answers 409 and leaves status, history and stream untouched
// (AC-05).
func TestSetStatusRejectsForbiddenTransitionWithoutChange(t *testing.T) {
	env := newStatusEnv(t)
	historyBefore := env.historyCount(t)

	rec := env.postStatus(t, model.StatusDone)
	if rec.Code != http.StatusConflict {
		t.Fatalf("angefragt -> fertig = %d, want 409 (body %q)", rec.Code, rec.Body.String())
	}
	var envl envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &envl); err != nil {
		t.Fatalf("decode error envelope: %v", err)
	}
	if envl.Error.Code != CodeConflict {
		t.Fatalf("error code = %q, want %q", envl.Error.Code, CodeConflict)
	}
	if got := env.currentStatus(t); got != model.StatusRequested {
		t.Fatalf("status after rejected transition = %q, want %q", got, model.StatusRequested)
	}
	if got := env.historyCount(t); got != historyBefore {
		t.Fatalf("history count after rejected transition = %d, want %d", got, historyBefore)
	}
	if msgs := env.orderMessages(t); len(msgs) != 0 {
		t.Fatalf("rejected transition published %d messages, want 0", len(msgs))
	}

	// A backward step is forbidden too: bestaetigt -> angefragt.
	if rec := env.postStatus(t, model.StatusConfirmed); rec.Code != http.StatusOK {
		t.Fatalf("angefragt -> bestaetigt = %d, want 200", rec.Code)
	}
	if rec := env.postStatus(t, model.StatusRequested); rec.Code != http.StatusConflict {
		t.Fatalf("bestaetigt -> angefragt = %d, want 409", rec.Code)
	}
	if got := env.currentStatus(t); got != model.StatusConfirmed {
		t.Fatalf("status = %q, want %q", got, model.StatusConfirmed)
	}
}

// TestSetStatusDonePublishesExactlyOneMessage proves that reaching 'fertig'
// appends exactly one {order_number,plate} message to the invoice stream and
// that a second 'fertig' is impossible and publishes nothing (AC-07).
func TestSetStatusDonePublishesExactlyOneMessage(t *testing.T) {
	env := newStatusEnv(t)

	if rec := env.postStatus(t, model.StatusConfirmed); rec.Code != http.StatusOK {
		t.Fatalf("angefragt -> bestaetigt = %d, want 200", rec.Code)
	}
	if rec := env.postStatus(t, model.StatusInProgress); rec.Code != http.StatusOK {
		t.Fatalf("bestaetigt -> in_arbeit = %d, want 200", rec.Code)
	}
	if rec := env.postStatus(t, model.StatusDone); rec.Code != http.StatusOK {
		t.Fatalf("in_arbeit -> fertig = %d, want 200", rec.Code)
	}

	msgs := env.orderMessages(t)
	if len(msgs) != 1 {
		t.Fatalf("invoice stream messages after fertig = %d, want 1", len(msgs))
	}
	if got := msgs[0].FieldValues["order_number"]; got != env.order {
		t.Fatalf("message order_number = %q, want %q", got, env.order)
	}
	if got := msgs[0].FieldValues["plate"]; got != env.plate {
		t.Fatalf("message plate = %q, want %q", got, env.plate)
	}

	// The status has already moved on, so a second 'fertig' is a conflict and
	// must not publish a second message.
	if rec := env.postStatus(t, model.StatusDone); rec.Code != http.StatusConflict {
		t.Fatalf("second in_arbeit -> fertig = %d, want 409", rec.Code)
	}
	if msgs := env.orderMessages(t); len(msgs) != 1 {
		t.Fatalf("invoice stream messages after second fertig = %d, want 1", len(msgs))
	}
}

// TestSetStatusRequiresEmployeeAndKnownStatus covers the input guards: an
// unauthenticated caller answers 401 and an unknown status value is a
// validation error.
func TestSetStatusRequiresEmployeeAndKnownStatus(t *testing.T) {
	env := newStatusEnv(t)

	req := httptest.NewRequest(http.MethodPost,
		"/api/workshop/orders/"+env.order+"/status", strings.NewReader(`{"status":"bestaetigt"}`))
	req.SetPathValue("order_number", env.order)
	rec := httptest.NewRecorder()
	handleSetStatus(env.store, env.pub).ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated = %d, want 401", rec.Code)
	}

	if rec := env.postStatus(t, "unknown"); rec.Code != http.StatusBadRequest {
		t.Fatalf("unknown status = %d, want 400 (body %q)", rec.Code, rec.Body.String())
	}
	if got := env.currentStatus(t); got != model.StatusRequested {
		t.Fatalf("status after invalid request = %q, want %q", got, model.StatusRequested)
	}
}
