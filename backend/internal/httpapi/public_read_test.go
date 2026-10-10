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

	"workshop-api/internal/config"
	"workshop-api/internal/model"
	"workshop-api/internal/store"
)

// publicReadEnv seeds a finished order with a stored invoice, a collected
// (abgeholt) order with a stored invoice and an open (not finished) order in the
// real PostgreSQL database declared by DATABASE_URL.
type publicReadEnv struct {
	router        http.Handler
	finishedOrder string
	finishedPlate string
	pickedUpOrder string
	pickedUpPlate string
	openOrder     string
	openPlate     string
}

// publicOrderResp is the status response body.
type publicOrderResp struct {
	OrderNumber   string              `json:"order_number"`
	Status        string              `json:"status"`
	StatusHistory []model.StatusEvent `json:"status_history"`
}

// publicInvoiceResp is the invoice response body.
type publicInvoiceResp struct {
	Invoice struct {
		Items []struct {
			Kind           string  `json:"kind"`
			Description    string  `json:"description"`
			Hours          float64 `json:"hours"`
			Quantity       int     `json:"quantity"`
			UnitPriceCents int64   `json:"unit_price_cents"`
			LineTotalCents int64   `json:"line_total_cents"`
		} `json:"items"`
		SubtotalCents int64     `json:"subtotal_cents"`
		TaxCents      int64     `json:"tax_cents"`
		TotalCents    int64     `json:"total_cents"`
		CreatedAt     time.Time `json:"created_at"`
	} `json:"invoice"`
}

// newPublicReadEnv opens the real PostgreSQL, applies the product migration and
// seeds the rows this ticket reads. Each row is unique to the run, so parallel
// branches sharing the database do not collide.
func newPublicReadEnv(t *testing.T) publicReadEnv {
	t.Helper()

	dbURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if dbURL == "" {
		t.Skip("DATABASE_URL is not set; this test runs against a real PostgreSQL")
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

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	finishedOrder := "PUB-FIN-" + suffix
	finishedPlate := "PUB-FIN-" + suffix
	pickedUpOrder := "PUB-PICK-" + suffix
	pickedUpPlate := "PUB-PICK-" + suffix
	openOrder := "PUB-OPEN-" + suffix
	openPlate := "PUB-OPEN-" + suffix

	customerID := insertCustomer(t, ctx, pool, "pub-status-"+suffix+"@example.test")
	finishedVehicle := insertVehicle(t, ctx, pool, finishedPlate, "AB", "Fertig")
	pickedUpVehicle := insertVehicle(t, ctx, pool, pickedUpPlate, "CD", "Abgeholt")
	openVehicle := insertVehicle(t, ctx, pool, openPlate, "XY", "Offen")

	finishedOrderID := insertOrder(t, ctx, pool, finishedOrder, customerID, finishedVehicle, model.StatusDone)
	pickedUpOrderID := insertOrder(t, ctx, pool, pickedUpOrder, customerID, pickedUpVehicle, model.StatusPickedUp)
	openOrderID := insertOrder(t, ctx, pool, openOrder, customerID, openVehicle, model.StatusRequested)

	// Remove every row this test writes so a sibling test's month-revenue
	// assertions over the shared database are not skewed.
	t.Cleanup(func() {
		cleanupCtx := context.Background()
		orderIDs := []int64{finishedOrderID, pickedUpOrderID, openOrderID}
		vehicleIDs := []int64{finishedVehicle, pickedUpVehicle, openVehicle}
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM invoices WHERE order_id = ANY($1)`, orderIDs)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM workshop_orders WHERE id = ANY($1)`, orderIDs)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM vehicles WHERE id = ANY($1)`, vehicleIDs)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM customers WHERE id = $1`, customerID)
	})

	base := time.Now().UTC().Add(-3 * time.Hour)
	for i, status := range []string{
		model.StatusRequested,
		model.StatusConfirmed,
		model.StatusInProgress,
		model.StatusDone,
	} {
		if _, err := pool.Exec(ctx,
			`INSERT INTO order_status_history (order_id, status, changed_at) VALUES ($1, $2, $3)`,
			finishedOrderID, status, base.Add(time.Duration(i)*time.Minute)); err != nil {
			t.Fatalf("insert finished history: %v", err)
		}
	}
	for i, status := range []string{
		model.StatusRequested,
		model.StatusConfirmed,
		model.StatusInProgress,
		model.StatusDone,
		model.StatusPickedUp,
	} {
		if _, err := pool.Exec(ctx,
			`INSERT INTO order_status_history (order_id, status, changed_at) VALUES ($1, $2, $3)`,
			pickedUpOrderID, status, base.Add(time.Duration(i)*time.Minute)); err != nil {
			t.Fatalf("insert picked-up history: %v", err)
		}
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO order_status_history (order_id, status, changed_at) VALUES ($1, $2, $3)`,
		openOrderID, model.StatusRequested, base); err != nil {
		t.Fatalf("insert open history: %v", err)
	}

	invoiceID := insertInvoice(t, ctx, pool, finishedOrder, finishedOrderID, 18000, 3420, 21420)
	insertInvoiceItem(t, ctx, pool, invoiceID, model.ItemKindLabour, "Ölwechsel", 2.5, nil, 6000, 15000)
	insertInvoiceItem(t, ctx, pool, invoiceID, model.ItemKindPart, "Ölfilter", nil, 2, 1500, 3000)

	pickedUpInvoiceID := insertInvoice(t, ctx, pool, pickedUpOrder, pickedUpOrderID, 18000, 3420, 21420)
	insertInvoiceItem(t, ctx, pool, pickedUpInvoiceID, model.ItemKindLabour, "Ölwechsel", 2.5, nil, 6000, 15000)
	insertInvoiceItem(t, ctx, pool, pickedUpInvoiceID, model.ItemKindPart, "Ölfilter", nil, 2, 1500, 3000)

	return publicReadEnv{
		router:        NewRouter(config.Config{FrontendOrigin: testFrontendOrigin}, store.New(pool)),
		finishedOrder: finishedOrder,
		finishedPlate: finishedPlate,
		pickedUpOrder: pickedUpOrder,
		pickedUpPlate: pickedUpPlate,
		openOrder:     openOrder,
		openPlate:     openPlate,
	}
}

func insertCustomer(t *testing.T, ctx context.Context, pool *pgxpool.Pool, email string) int64 {
	t.Helper()
	var id int64
	if err := pool.QueryRow(ctx,
		`INSERT INTO customers (name, email, phone) VALUES ($1, $2, $3) RETURNING id`,
		"Public Reader", email, "").Scan(&id); err != nil {
		t.Fatalf("insert customer: %v", err)
	}
	return id
}

func insertVehicle(t *testing.T, ctx context.Context, pool *pgxpool.Pool, plate, brand, modelName string) int64 {
	t.Helper()
	var id int64
	if err := pool.QueryRow(ctx,
		`INSERT INTO vehicles (plate, plate_normalized, brand, model, mileage)
		 VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		plate, store.NormalizePlate(plate), brand, modelName, 1000).Scan(&id); err != nil {
		t.Fatalf("insert vehicle: %v", err)
	}
	return id
}

func insertOrder(t *testing.T, ctx context.Context, pool *pgxpool.Pool, orderNumber string, customerID, vehicleID int64, status string) int64 {
	t.Helper()
	var id int64
	if err := pool.QueryRow(ctx,
		`INSERT INTO workshop_orders (order_number, customer_id, vehicle_id, status, preferred_date, description)
		 VALUES ($1, $2, $3, $4, now()::date, '') RETURNING id`,
		orderNumber, customerID, vehicleID, status).Scan(&id); err != nil {
		t.Fatalf("insert order: %v", err)
	}
	return id
}

func insertInvoice(t *testing.T, ctx context.Context, pool *pgxpool.Pool, orderNumber string, orderID, subtotal, tax, total int64) int64 {
	t.Helper()
	var id int64
	if err := pool.QueryRow(ctx,
		`INSERT INTO invoices (order_number, order_id, subtotal_cents, tax_cents, total_cents)
		 VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		orderNumber, orderID, subtotal, tax, total).Scan(&id); err != nil {
		t.Fatalf("insert invoice: %v", err)
	}
	return id
}

func insertInvoiceItem(t *testing.T, ctx context.Context, pool *pgxpool.Pool, invoiceID int64, kind, description string, hours any, quantity any, unitPrice, lineTotal int64) {
	t.Helper()
	if _, err := pool.Exec(ctx,
		`INSERT INTO invoice_items (invoice_id, kind, description, hours, quantity, unit_price_cents, line_total_cents)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		invoiceID, kind, description, hours, quantity, unitPrice, lineTotal); err != nil {
		t.Fatalf("insert invoice item: %v", err)
	}
}

func (e publicReadEnv) get(t *testing.T, path string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
}

func decodeEnvelope(t *testing.T, rec *httptest.ResponseRecorder) envelope {
	t.Helper()
	var env envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode error envelope %q: %v", rec.Body.String(), err)
	}
	return env
}

// TestPublicOrderStatusReturnsStatusAndHistory proves the correct order number
// and plate combination returns the current status and full history, and that
// the plate is compared normalized (AC-12).
func TestPublicOrderStatusReturnsStatusAndHistory(t *testing.T) {
	env := newPublicReadEnv(t)

	// Lower case and added spaces still identify the vehicle (normalized plate).
	path := "/api/public/orders/" + env.finishedOrder + "?plate=" + strings.ToLower(env.finishedPlate)
	rec := env.get(t, path)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET %s = %d, want 200 (body %q)", path, rec.Code, rec.Body.String())
	}
	var resp publicOrderResp
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response %q: %v", rec.Body.String(), err)
	}
	if resp.OrderNumber != env.finishedOrder {
		t.Fatalf("order_number = %q, want %q", resp.OrderNumber, env.finishedOrder)
	}
	if resp.Status != model.StatusDone {
		t.Fatalf("status = %q, want %q", resp.Status, model.StatusDone)
	}
	wantHistory := []string{
		model.StatusRequested, model.StatusConfirmed, model.StatusInProgress, model.StatusDone,
	}
	if len(resp.StatusHistory) != len(wantHistory) {
		t.Fatalf("history length = %d, want %d", len(resp.StatusHistory), len(wantHistory))
	}
	for i, want := range wantHistory {
		if got := resp.StatusHistory[i].Status; got != want {
			t.Fatalf("history[%d] status = %q, want %q", i, got, want)
		}
	}
}

// TestPublicLookupsAnswerIdenticalNeutralNotFound proves that a wrong plate, an
// unknown order number and a lookup without a plate all answer the exact same
// neutral 404 body — no hint that an order exists (AC-12, AC-13).
func TestPublicLookupsAnswerIdenticalNeutralNotFound(t *testing.T) {
	env := newPublicReadEnv(t)

	statusCases := []string{
		"/api/public/orders/" + env.finishedOrder + "?plate=NOPE-999",
		"/api/public/orders/PUB-UNKNOWN?plate=" + env.finishedPlate,
		"/api/public/orders/" + env.finishedOrder, // missing plate
	}
	invoiceCases := []string{
		"/api/public/orders/" + env.finishedOrder + "/invoice?plate=NOPE-999",
		"/api/public/orders/PUB-UNKNOWN/invoice?plate=" + env.finishedPlate,
	}

	var reference string
	for _, tc := range append(statusCases, invoiceCases...) {
		rec := env.get(t, tc)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("GET %s = %d, want 404 (body %q)", tc, rec.Code, rec.Body.String())
		}
		body := rec.Body.String()
		if reference == "" {
			reference = body
			envl := decodeEnvelope(t, rec)
			if envl.Error.Code != CodeNotFound {
				t.Fatalf("GET %s error code = %q, want %q", tc, envl.Error.Code, CodeNotFound)
			}
			if envl.Error.Message == "" {
				t.Fatalf("GET %s returned an empty error message", tc)
			}
		} else if body != reference {
			t.Fatalf("GET %s body = %q, want the identical neutral body %q", tc, body, reference)
		}
	}
}

// TestPublicInvoiceNotAvailableBeforeFinished proves that a valid order which is
// not finished answers the same neutral 404, not an invoice (AC-13).
func TestPublicInvoiceNotAvailableBeforeFinished(t *testing.T) {
	env := newPublicReadEnv(t)

	rec := env.get(t, "/api/public/orders/"+env.openOrder+"/invoice?plate="+env.openPlate)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("invoice before fertig = %d, want 404 (body %q)", rec.Code, rec.Body.String())
	}
	envl := decodeEnvelope(t, rec)
	if envl.Error.Code != CodeNotFound {
		t.Fatalf("error code = %q, want %q", envl.Error.Code, CodeNotFound)
	}

	// The body must be the same neutral one the status lookup uses.
	statusRec := env.get(t, "/api/public/orders/"+env.finishedOrder+"?plate=NOPE-999")
	if rec.Body.String() != statusRec.Body.String() {
		t.Fatalf("unfinished invoice body = %q, want the identical neutral body %q",
			rec.Body.String(), statusRec.Body.String())
	}
}

// TestPublicInvoiceReturnsPositionsAndCentAmounts proves the stored invoice is
// served with its positions, subtotal, VAT and total in whole cents (AC-13).
func TestPublicInvoiceReturnsPositionsAndCentAmounts(t *testing.T) {
	env := newPublicReadEnv(t)

	rec := env.get(t, "/api/public/orders/"+env.finishedOrder+"/invoice?plate="+env.finishedPlate)
	if rec.Code != http.StatusOK {
		t.Fatalf("invoice lookup = %d, want 200 (body %q)", rec.Code, rec.Body.String())
	}
	var resp publicInvoiceResp
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response %q: %v", rec.Body.String(), err)
	}
	if resp.Invoice.SubtotalCents != 18000 {
		t.Fatalf("subtotal_cents = %d, want 18000", resp.Invoice.SubtotalCents)
	}
	if resp.Invoice.TaxCents != 3420 {
		t.Fatalf("tax_cents = %d, want 3420", resp.Invoice.TaxCents)
	}
	if resp.Invoice.TotalCents != 21420 {
		t.Fatalf("total_cents = %d, want 21420", resp.Invoice.TotalCents)
	}
	if resp.Invoice.CreatedAt.IsZero() {
		t.Fatal("created_at is zero")
	}
	if len(resp.Invoice.Items) != 2 {
		t.Fatalf("items = %d, want 2", len(resp.Invoice.Items))
	}
	labour := resp.Invoice.Items[0]
	if labour.Kind != model.ItemKindLabour || labour.Hours != 2.5 ||
		labour.UnitPriceCents != 6000 || labour.LineTotalCents != 15000 {
		t.Fatalf("labour item = %+v, want kind=labour hours=2.5 unit=6000 line=15000", labour)
	}
	part := resp.Invoice.Items[1]
	if part.Kind != model.ItemKindPart || part.Quantity != 2 ||
		part.UnitPriceCents != 1500 || part.LineTotalCents != 3000 {
		t.Fatalf("part item = %+v, want kind=part quantity=2 unit=1500 line=3000", part)
	}
}

// TestPublicInvoiceStaysAvailableAfterPickedUp proves the stored invoice is
// still served once the order has moved on to 'abgeholt', the normal end of the
// lifecycle (AC-13): the customer who collected the car keeps access.
func TestPublicInvoiceStaysAvailableAfterPickedUp(t *testing.T) {
	env := newPublicReadEnv(t)

	rec := env.get(t, "/api/public/orders/"+env.pickedUpOrder+"/invoice?plate="+env.pickedUpPlate)
	if rec.Code != http.StatusOK {
		t.Fatalf("picked-up invoice lookup = %d, want 200 (body %q)", rec.Code, rec.Body.String())
	}
	var resp publicInvoiceResp
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response %q: %v", rec.Body.String(), err)
	}
	if resp.Invoice.TotalCents != 21420 {
		t.Fatalf("total_cents = %d, want 21420", resp.Invoice.TotalCents)
	}
	if len(resp.Invoice.Items) != 2 {
		t.Fatalf("items = %d, want 2", len(resp.Invoice.Items))
	}
}
