package httpapi

import (
	"context"
	"encoding/json"
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

// workshopOrdersPool opens the real PostgreSQL database the API runs on and
// applies the product's own migration. No SQLite, no in-memory stand-in.
func workshopOrdersPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	url := os.Getenv("DATABASE_URL")
	if strings.TrimSpace(url) == "" {
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
	return pool
}

type seedOrderSpec struct {
	orderNumber   string
	status        string
	plate         string
	preferredDate *time.Time
	customerName  string
	items         []model.OrderItem
	history       []string
}

// seedWorkshopOrder inserts a customer, a vehicle, an order and its positions
// and history. It removes exactly those rows again when the test ends, so it
// never touches data another ticket owns.
func seedWorkshopOrder(t *testing.T, pool *pgxpool.Pool, spec seedOrderSpec) {
	t.Helper()
	ctx := context.Background()

	email := strings.ToLower(spec.orderNumber) + "@zzordersslice.example"
	var customerID int64
	if err := pool.QueryRow(ctx,
		`INSERT INTO customers (name, email, phone) VALUES ($1, $2, $3) RETURNING id`,
		spec.customerName, email, "0000-000000").Scan(&customerID); err != nil {
		t.Fatalf("seed customer: %v", err)
	}

	normalized := store.NormalizePlate(spec.plate)
	var vehicleID int64
	if err := pool.QueryRow(ctx,
		`INSERT INTO vehicles (plate, plate_normalized, brand, model, mileage) VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		spec.plate, normalized, "Testmarke", "Testmodell", 12345).Scan(&vehicleID); err != nil {
		t.Fatalf("seed vehicle: %v", err)
	}

	var orderID int64
	if err := pool.QueryRow(ctx,
		`INSERT INTO workshop_orders (order_number, customer_id, vehicle_id, status, preferred_date, description)
		 VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
		spec.orderNumber, customerID, vehicleID, spec.status, spec.preferredDate, "Testauftrag").Scan(&orderID); err != nil {
		t.Fatalf("seed order: %v", err)
	}

	for _, item := range spec.items {
		if _, err := pool.Exec(ctx,
			`INSERT INTO order_items (order_id, kind, description, hours, quantity, unit_price_cents, line_total_cents)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			orderID, item.Kind, item.Description, item.Hours, item.Quantity, item.UnitPriceCents, item.LineTotalCents); err != nil {
			t.Fatalf("seed order item: %v", err)
		}
	}

	for _, status := range spec.history {
		if _, err := pool.Exec(ctx,
			`INSERT INTO order_status_history (order_id, status) VALUES ($1, $2)`, orderID, status); err != nil {
			t.Fatalf("seed status history: %v", err)
		}
	}

	t.Cleanup(func() {
		cleanupCtx := context.Background()
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM workshop_orders WHERE order_number = $1`, spec.orderNumber)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM customers WHERE email = $1`, email)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM vehicles WHERE plate_normalized = $1`, normalized)
	})
}

// callOrderHandler invokes an order handler with the signed-in employee placed
// in the context via WithEmployee and a Bearer header, exactly the seam the
// guard middleware will fill in once it merges.
func callOrderHandler(handler http.HandlerFunc, method, target, orderNumber string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, nil)
	req = req.WithContext(WithEmployee(req.Context(), model.Employee{ID: 1, Email: "chef@werkstatt.example"}))
	req.Header.Set("Authorization", "Bearer test-token")
	if orderNumber != "" {
		req.SetPathValue("order_number", orderNumber)
	}
	rec := httptest.NewRecorder()
	handler(rec, req)
	return rec
}

func dateUTC(year int, month time.Month, day int) *time.Time {
	d := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
	return &d
}

type orderListResponse struct {
	Orders []model.OrderRow `json:"orders"`
}

type orderDetailResponse struct {
	Order model.OrderDetail `json:"order"`
}

func decodeOrderList(t *testing.T, rec *httptest.ResponseRecorder) []model.OrderRow {
	t.Helper()
	var body orderListResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode list body %q: %v", rec.Body.String(), err)
	}
	return body.Orders
}

// TestListOrdersFiltersAndSearches proves that the status filter and the
// case-insensitive plate substring select exactly the matching orders and that
// the result is ordered by preferred date then order number.
func TestListOrdersFiltersAndSearches(t *testing.T) {
	pool := workshopOrdersPool(t)

	seedWorkshopOrder(t, pool, seedOrderSpec{
		orderNumber: "ZZWLIST-B", status: model.StatusRequested, plate: "ZZWLIST-B",
		preferredDate: dateUTC(2026, 3, 1), customerName: "Anna Angefragt",
		history: []string{model.StatusRequested},
	})
	seedWorkshopOrder(t, pool, seedOrderSpec{
		orderNumber: "ZZWLIST-A", status: model.StatusRequested, plate: "ZZWLIST-A",
		preferredDate: dateUTC(2026, 3, 2), customerName: "Berta Angefragt",
		history: []string{model.StatusRequested},
	})
	seedWorkshopOrder(t, pool, seedOrderSpec{
		orderNumber: "ZZWLIST-C", status: model.StatusConfirmed, plate: "ZZWLIST-C",
		preferredDate: dateUTC(2026, 3, 3), customerName: "Clara Bestaetigt",
		history: []string{model.StatusRequested, model.StatusConfirmed},
	})

	handler := handleListOrders(store.New(pool))

	// Search is a case-insensitive substring and yields exactly the three
	// seeded orders, sorted by preferred date.
	rec := callOrderHandler(handler, http.MethodGet, "/api/workshop/orders?plate=zzwlist", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("list = %d, want 200 (body %q)", rec.Code, rec.Body.String())
	}
	all := decodeOrderList(t, rec)
	if len(all) != 3 {
		t.Fatalf("plate search hit count = %d, want 3 (body %q)", len(all), rec.Body.String())
	}
	wantOrder := []string{"ZZWLIST-B", "ZZWLIST-A", "ZZWLIST-C"}
	for i, want := range wantOrder {
		if all[i].OrderNumber != want {
			t.Fatalf("order[%d] = %q, want %q (sorted by preferred date)", i, all[i].OrderNumber, want)
		}
	}

	// The status filter narrows the same plate search.
	rec = callOrderHandler(handler, http.MethodGet, "/api/workshop/orders?plate=zzwlist&status=angefragt", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("filtered list = %d, want 200 (body %q)", rec.Code, rec.Body.String())
	}
	requested := decodeOrderList(t, rec)
	if len(requested) != 2 {
		t.Fatalf("angefragt hit count = %d, want 2", len(requested))
	}
	for _, row := range requested {
		if row.Status != model.StatusRequested {
			t.Fatalf("filtered row status = %q, want %q", row.Status, model.StatusRequested)
		}
	}

	rec = callOrderHandler(handler, http.MethodGet, "/api/workshop/orders?plate=ZZWLIST&status=bestaetigt", "")
	if got := decodeOrderList(t, rec); len(got) != 1 {
		t.Fatalf("bestaetigt hit count = %d, want 1", len(got))
	}

	rec = callOrderHandler(handler, http.MethodGet, "/api/workshop/orders?plate=zzwlist&status=fertig", "")
	if got := decodeOrderList(t, rec); len(got) != 0 {
		t.Fatalf("fertig hit count = %d, want 0", len(got))
	}
}

// TestListOrdersCarriesTotalAndCustomerName proves the row carries the summed
// position total and the customer name.
func TestListOrdersCarriesTotalAndCustomerName(t *testing.T) {
	pool := workshopOrdersPool(t)

	seedWorkshopOrder(t, pool, seedOrderSpec{
		orderNumber: "ZZWLIST-TOT", status: model.StatusConfirmed, plate: "ZZWLIST-TOT",
		preferredDate: dateUTC(2026, 4, 1), customerName: "Dora Summe",
		items: []model.OrderItem{
			{Kind: model.ItemKindLabour, Description: "Arbeit", Hours: 1.5, UnitPriceCents: 9000, LineTotalCents: 13500},
			{Kind: model.ItemKindPart, Description: "Teil", Quantity: 2, UnitPriceCents: 2500, LineTotalCents: 5000},
		},
		history: []string{model.StatusRequested, model.StatusConfirmed},
	})

	handler := handleListOrders(store.New(pool))
	rec := callOrderHandler(handler, http.MethodGet, "/api/workshop/orders?plate=zzwlist-tot", "")
	rows := decodeOrderList(t, rec)
	if len(rows) != 1 {
		t.Fatalf("hit count = %d, want 1 (body %q)", len(rows), rec.Body.String())
	}
	row := rows[0]
	if row.TotalCents != 18500 {
		t.Fatalf("total_cents = %d, want 18500", row.TotalCents)
	}
	if row.CustomerName != "Dora Summe" {
		t.Fatalf("customer_name = %q, want %q", row.CustomerName, "Dora Summe")
	}
	if row.Plate != "ZZWLIST-TOT" {
		t.Fatalf("plate = %q, want %q", row.Plate, "ZZWLIST-TOT")
	}
	if row.PreferredDate != "2026-04-01" {
		t.Fatalf("preferred_date = %q, want 2026-04-01", row.PreferredDate)
	}
}

// TestOrderDetailCarriesPositionsAndHistory proves the detail carries customer,
// vehicle, positions, status history and the subtotal, and that an unknown order
// number answers 404 not_found.
func TestOrderDetailCarriesPositionsAndHistory(t *testing.T) {
	pool := workshopOrdersPool(t)

	seedWorkshopOrder(t, pool, seedOrderSpec{
		orderNumber: "ZZWLIST-DET", status: model.StatusInProgress, plate: "ZZWLIST-DET",
		preferredDate: dateUTC(2026, 5, 1), customerName: "Emil Detail",
		items: []model.OrderItem{
			{Kind: model.ItemKindLabour, Description: "Oelwechsel", Hours: 1.5, UnitPriceCents: 9000, LineTotalCents: 13500},
			{Kind: model.ItemKindPart, Description: "Oelfilter", Quantity: 2, UnitPriceCents: 500, LineTotalCents: 1000},
		},
		history: []string{model.StatusRequested, model.StatusConfirmed, model.StatusInProgress},
	})

	handler := handleOrderDetail(store.New(pool))
	rec := callOrderHandler(handler, http.MethodGet, "/api/workshop/orders/ZZWLIST-DET", "ZZWLIST-DET")
	if rec.Code != http.StatusOK {
		t.Fatalf("detail = %d, want 200 (body %q)", rec.Code, rec.Body.String())
	}
	var body orderDetailResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode detail: %v", err)
	}
	order := body.Order
	if len(order.Items) != 2 {
		t.Fatalf("items = %d, want 2", len(order.Items))
	}
	if len(order.StatusHistory) != 3 {
		t.Fatalf("status history = %d, want 3", len(order.StatusHistory))
	}
	if order.SubtotalCents != 14500 {
		t.Fatalf("subtotal_cents = %d, want 14500", order.SubtotalCents)
	}
	if order.TotalCents != 14500 {
		t.Fatalf("total_cents = %d, want 14500", order.TotalCents)
	}
	if order.Customer.Name != "Emil Detail" || order.Customer.Email == "" {
		t.Fatalf("customer = %+v, want name and email", order.Customer)
	}
	if order.Vehicle.Plate != "ZZWLIST-DET" {
		t.Fatalf("vehicle plate = %q, want ZZWLIST-DET", order.Vehicle.Plate)
	}
	if order.StatusHistory[0].Status != model.StatusRequested || order.StatusHistory[2].Status != model.StatusInProgress {
		t.Fatalf("status history = %+v, want angefragt..in_arbeit", order.StatusHistory)
	}

	// Unknown order number -> 404 not_found.
	rec = callOrderHandler(handler, http.MethodGet, "/api/workshop/orders/ZZWLIST-MISSING", "ZZWLIST-MISSING")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("unknown detail = %d, want 404", rec.Code)
	}
	var env envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode 404 body: %v", err)
	}
	if env.Error.Code != CodeNotFound {
		t.Fatalf("error code = %q, want %q", env.Error.Code, CodeNotFound)
	}
}

// TestListOrdersRejectsUnknownStatus proves the status filter is validated.
func TestListOrdersRejectsUnknownStatus(t *testing.T) {
	pool := workshopOrdersPool(t)
	handler := handleListOrders(store.New(pool))

	rec := callOrderHandler(handler, http.MethodGet, "/api/workshop/orders?status=bogus", "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid status = %d, want 400 (body %q)", rec.Code, rec.Body.String())
	}
	var env envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if env.Error.Code != CodeValidationFailed {
		t.Fatalf("error code = %q, want %q", env.Error.Code, CodeValidationFailed)
	}
}
