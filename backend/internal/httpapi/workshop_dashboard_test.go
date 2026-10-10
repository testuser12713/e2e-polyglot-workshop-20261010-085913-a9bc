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

	"workshop-api/internal/model"
	"workshop-api/internal/store"
)

// dashboardEnv is the real PostgreSQL store of the dashboard tests plus a
// per-run suffix that keeps every seeded row unique (AC-27).
type dashboardEnv struct {
	pool   *pgxpool.Pool
	store  *store.Store
	suffix string
}

// dashboardBody decodes GET /api/workshop/dashboard.
type dashboardBody struct {
	Open              []model.OrderRow `json:"open"`
	FinishedToday     []model.OrderRow `json:"finished_today"`
	RevenueMonthCents int64            `json:"revenue_month_cents"`
}

// newDashboardEnv connects to the real PostgreSQL declared by DATABASE_URL,
// applies the product migration and returns the store. There is no SQLite or
// in-memory stand-in (AC-27).
func newDashboardEnv(t *testing.T) dashboardEnv {
	t.Helper()

	url := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if url == "" {
		t.Skip("DATABASE_URL is not set; this test runs against a real PostgreSQL")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatalf("connect to PostgreSQL: %v", err)
	}
	t.Cleanup(pool.Close)
	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("ping PostgreSQL: %v", err)
	}
	applyTestMigration(t, ctx, pool)

	return dashboardEnv{
		pool:   pool,
		store:  store.New(pool),
		suffix: fmt.Sprintf("%d", time.Now().UnixNano()),
	}
}

// seedOrder creates a customer, a vehicle and an order in the given status and
// returns the order's number, plate, customer name and id.
func (e dashboardEnv) seedOrder(t *testing.T, status string) (orderNumber, plate, customerName string, orderID int64) {
	t.Helper()
	ctx := context.Background()

	tag := fmt.Sprintf("%s-%s-%d", e.suffix, status, time.Now().UnixNano())
	customerName = "Dashboard " + tag
	email := "dashboard-" + tag + "@example.test"
	plate = "B-DASH-" + tag

	var customerID, vehicleID int64
	if err := e.pool.QueryRow(ctx,
		`INSERT INTO customers (name, email, phone) VALUES ($1, $2, $3) RETURNING id`,
		customerName, email, "").Scan(&customerID); err != nil {
		t.Fatalf("insert customer: %v", err)
	}
	if err := e.pool.QueryRow(ctx,
		`INSERT INTO vehicles (plate, plate_normalized, brand, model, mileage)
		 VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		plate, strings.ToUpper(plate), "VW", "Golf", 1000).Scan(&vehicleID); err != nil {
		t.Fatalf("insert vehicle: %v", err)
	}

	orderNumber = "DASH-" + tag
	if err := e.pool.QueryRow(ctx,
		`INSERT INTO workshop_orders (order_number, customer_id, vehicle_id, status, preferred_date, description)
		 VALUES ($1, $2, $3, $4, now()::date, '') RETURNING id`,
		orderNumber, customerID, vehicleID, status).Scan(&orderID); err != nil {
		t.Fatalf("insert order: %v", err)
	}
	return orderNumber, plate, customerName, orderID
}

// setOrderStatus updates an order's status directly.
func (e dashboardEnv) setOrderStatus(t *testing.T, orderID int64, status string) {
	t.Helper()
	if _, err := e.pool.Exec(context.Background(),
		`UPDATE workshop_orders SET status = $1 WHERE id = $2`, status, orderID); err != nil {
		t.Fatalf("update order status: %v", err)
	}
}

// addHistory appends a status history entry with an explicit UTC timestamp.
func (e dashboardEnv) addHistory(t *testing.T, orderID int64, status string, changedAt time.Time) {
	t.Helper()
	if _, err := e.pool.Exec(context.Background(),
		`INSERT INTO order_status_history (order_id, status, changed_at) VALUES ($1, $2, $3)`,
		orderID, status, changedAt.UTC()); err != nil {
		t.Fatalf("insert status history: %v", err)
	}
}

// addItem appends an order position with the given line total in cents.
func (e dashboardEnv) addItem(t *testing.T, orderID int64, lineTotalCents int64) {
	t.Helper()
	if _, err := e.pool.Exec(context.Background(),
		`INSERT INTO order_items (order_id, kind, description, quantity, unit_price_cents, line_total_cents)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		orderID, model.ItemKindPart, "", 1, lineTotalCents, lineTotalCents); err != nil {
		t.Fatalf("insert order item: %v", err)
	}
}

// addInvoice records an invoice with an explicit total in cents and creation
// timestamp.
func (e dashboardEnv) addInvoice(t *testing.T, orderNumber string, orderID int64, totalCents int64, createdAt time.Time) {
	t.Helper()
	if _, err := e.pool.Exec(context.Background(),
		`INSERT INTO invoices (order_number, order_id, subtotal_cents, tax_cents, total_cents, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		orderNumber, orderID, totalCents, 0, totalCents, createdAt.UTC()); err != nil {
		t.Fatalf("insert invoice: %v", err)
	}
}

// getDashboard calls the handler directly with the employee in the request
// context via WithEmployee, exactly as the auth middleware will.
func (e dashboardEnv) getDashboard(t *testing.T, withEmployee bool) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/workshop/dashboard", nil)
	if withEmployee {
		req = req.WithContext(WithEmployee(req.Context(), model.Employee{ID: 1, Email: "chef@werkstatt.example"}))
	}
	rec := httptest.NewRecorder()
	handleDashboard(e.store).ServeHTTP(rec, req)
	return rec
}

func decodeDashboard(t *testing.T, rec *httptest.ResponseRecorder) dashboardBody {
	t.Helper()
	var body dashboardBody
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode dashboard %q: %v", rec.Body.String(), err)
	}
	return body
}

func findRow(rows []model.OrderRow, orderNumber string) (model.OrderRow, bool) {
	for _, row := range rows {
		if row.OrderNumber == orderNumber {
			return row, true
		}
	}
	return model.OrderRow{}, false
}

// TestDashboardReportsOpenOrderWithTotalCents asserts that an order in one of
// the open statuses appears in the open list with plate, customer name and the
// summed line totals in whole cents (AC-21).
func TestDashboardReportsOpenOrderWithTotalCents(t *testing.T) {
	env := newDashboardEnv(t)

	orderNumber, plate, customerName, orderID := env.seedOrder(t, model.StatusInProgress)
	env.addItem(t, orderID, 1000)
	env.addItem(t, orderID, 250)

	rec := env.getDashboard(t, true)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET dashboard = %d, want 200 (body %q)", rec.Code, rec.Body.String())
	}
	body := decodeDashboard(t, rec)

	row, ok := findRow(body.Open, orderNumber)
	if !ok {
		t.Fatalf("open order %q missing from open list", orderNumber)
	}
	if row.Status != model.StatusInProgress {
		t.Fatalf("open order status = %q, want %q", row.Status, model.StatusInProgress)
	}
	if row.Plate != plate {
		t.Fatalf("open order plate = %q, want %q", row.Plate, plate)
	}
	if row.CustomerName != customerName {
		t.Fatalf("open order customer_name = %q, want %q", row.CustomerName, customerName)
	}
	if row.TotalCents != 1250 {
		t.Fatalf("open order total_cents = %d, want 1250", row.TotalCents)
	}
	if _, ok := findRow(body.FinishedToday, orderNumber); ok {
		t.Fatalf("open order %q must not appear in finished_today", orderNumber)
	}
}

// TestDashboardMovesOrderFromOpenToFinishedToday proves the numbers change the
// moment an order becomes 'fertig': it leaves the open list and enters the list
// of orders finished today (AC-21).
func TestDashboardMovesOrderFromOpenToFinishedToday(t *testing.T) {
	env := newDashboardEnv(t)

	orderNumber, _, _, orderID := env.seedOrder(t, model.StatusRequested)

	before := decodeDashboard(t, env.getDashboard(t, true))
	if _, ok := findRow(before.Open, orderNumber); !ok {
		t.Fatalf("order %q is not open before it is finished", orderNumber)
	}
	if _, ok := findRow(before.FinishedToday, orderNumber); ok {
		t.Fatalf("order %q is already finished before it is finished", orderNumber)
	}

	env.setOrderStatus(t, orderID, model.StatusDone)
	env.addHistory(t, orderID, model.StatusDone, time.Now().UTC())

	after := decodeDashboard(t, env.getDashboard(t, true))
	if _, ok := findRow(after.Open, orderNumber); ok {
		t.Fatalf("order %q stayed in the open list after being finished", orderNumber)
	}
	if _, ok := findRow(after.FinishedToday, orderNumber); !ok {
		t.Fatalf("order %q is missing from finished_today after being finished", orderNumber)
	}
}

// TestDashboardFinishedTodayUsesUTCDate asserts that only orders whose
// 'fertig' history entry falls on today's UTC date are listed (AC-21).
func TestDashboardFinishedTodayUsesUTCDate(t *testing.T) {
	env := newDashboardEnv(t)

	todayNumber, _, _, todayID := env.seedOrder(t, model.StatusDone)
	env.addHistory(t, todayID, model.StatusDone, time.Now().UTC())

	oldNumber, _, _, oldID := env.seedOrder(t, model.StatusDone)
	env.addHistory(t, oldID, model.StatusDone, time.Now().UTC().Add(-30*time.Hour))

	body := decodeDashboard(t, env.getDashboard(t, true))

	if _, ok := findRow(body.FinishedToday, todayNumber); !ok {
		t.Fatalf("order %q finished today is missing from finished_today", todayNumber)
	}
	if _, ok := findRow(body.FinishedToday, oldNumber); ok {
		t.Fatalf("order %q finished yesterday must not appear in finished_today", oldNumber)
	}
}

// TestDashboardRevenueSumsCurrentMonthInCents asserts that the revenue is the
// sum of the current month's invoices in whole cents and that a previous-month
// invoice is not counted (AC-21).
func TestDashboardRevenueSumsCurrentMonthInCents(t *testing.T) {
	env := newDashboardEnv(t)

	_, _, _, orderID := env.seedOrder(t, model.StatusDone)

	before := decodeDashboard(t, env.getDashboard(t, true)).RevenueMonthCents

	const currentMonthCents = int64(12345)
	const previousMonthCents = int64(999999999)

	env.addInvoice(t, "DASH-INV-NOW-"+env.suffix, orderID, currentMonthCents, time.Now().UTC())
	env.addInvoice(t, "DASH-INV-PREV-"+env.suffix, orderID, previousMonthCents, time.Now().UTC().AddDate(0, -1, 0))

	after := decodeDashboard(t, env.getDashboard(t, true)).RevenueMonthCents
	delta := after - before

	if delta < currentMonthCents {
		t.Fatalf("revenue delta = %d, want at least %d (current month invoice not summed)", delta, currentMonthCents)
	}
	// A previous-month amount this large is never part of the current month, so
	// if it leaked in the delta would jump by roughly previousMonthCents.
	if delta >= currentMonthCents+previousMonthCents {
		t.Fatalf("revenue delta = %d, previous-month invoice seems to be counted", delta)
	}
}

// TestDashboardRequiresEmployee asserts the handler rejects an unauthenticated
// caller before touching the database (AC-16).
func TestDashboardRequiresEmployee(t *testing.T) {
	env := newDashboardEnv(t)

	rec := env.getDashboard(t, false)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated dashboard = %d, want 401", rec.Code)
	}
	var envl envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &envl); err != nil {
		t.Fatalf("decode error envelope: %v", err)
	}
	if envl.Error.Code != CodeUnauthorized {
		t.Fatalf("error code = %q, want %q", envl.Error.Code, CodeUnauthorized)
	}
}
