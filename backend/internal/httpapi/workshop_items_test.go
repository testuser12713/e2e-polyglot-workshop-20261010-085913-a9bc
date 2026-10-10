package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"workshop-api/internal/config"
	"workshop-api/internal/model"
	"workshop-api/internal/store"
)

var itemSuffixCounter int64

func uniqueItemSuffix() string {
	n := atomic.AddInt64(&itemSuffixCounter, 1)
	return fmt.Sprintf("%d%d", time.Now().UnixNano(), n)
}

// newItemsTestEnv builds the real router against the real PostgreSQL database
// from DATABASE_URL and applies the product's own migration.
func newItemsTestEnv(t *testing.T) (*pgxpool.Pool, http.Handler) {
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

	router := NewRouter(config.Config{FrontendOrigin: testFrontendOrigin}, store.New(pool))
	return pool, router
}

// seedOrder inserts a customer, a vehicle and one workshop order in the given
// status and returns its order number.
func seedOrder(t *testing.T, pool *pgxpool.Pool, status string) string {
	t.Helper()
	ctx := context.Background()
	suffix := uniqueItemSuffix()

	var customerID int64
	if err := pool.QueryRow(ctx,
		`INSERT INTO customers (name, email, phone) VALUES ($1, $2, $3) RETURNING id`,
		"Test Kunde", "kunde-"+suffix+"@example.test", "0123456",
	).Scan(&customerID); err != nil {
		t.Fatalf("insert customer: %v", err)
	}

	plate := "B-" + suffix
	var vehicleID int64
	if err := pool.QueryRow(ctx,
		`INSERT INTO vehicles (plate, plate_normalized, brand, model, mileage)
		 VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		plate, strings.ToUpper(plate), "VW", "Golf", 120000,
	).Scan(&vehicleID); err != nil {
		t.Fatalf("insert vehicle: %v", err)
	}

	orderNumber := "AUF-" + suffix
	if _, err := pool.Exec(ctx,
		`INSERT INTO workshop_orders (order_number, customer_id, vehicle_id, status, description)
		 VALUES ($1, $2, $3, $4, $5)`,
		orderNumber, customerID, vehicleID, status, "Testauftrag",
	); err != nil {
		t.Fatalf("insert order: %v", err)
	}
	return orderNumber
}

// doItemRequest performs a workshop item request with the employee already in
// the request context (httpapi.WithEmployee).
func doItemRequest(t *testing.T, router http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, reader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	req = req.WithContext(WithEmployee(req.Context(), model.Employee{ID: 1, Email: "werkstatt@example.test"}))

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func decodeItemResponse(t *testing.T, rec *httptest.ResponseRecorder) model.OrderItem {
	t.Helper()
	var body struct {
		Item model.OrderItem `json:"item"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode item %q: %v", rec.Body.String(), err)
	}
	return body.Item
}

func itemPath(orderNumber string) string {
	return "/api/workshop/orders/" + orderNumber + "/items"
}

func itemPathWithID(orderNumber string, itemID int64) string {
	return fmt.Sprintf("/api/workshop/orders/%s/items/%d", orderNumber, itemID)
}

func createItem(t *testing.T, router http.Handler, orderNumber, body string) model.OrderItem {
	t.Helper()
	rec := doItemRequest(t, router, http.MethodPost, itemPath(orderNumber), body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST item = %d, want 201 (body %q)", rec.Code, rec.Body.String())
	}
	return decodeItemResponse(t, rec)
}

func decodeErrorCode(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var env envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode error %q: %v", rec.Body.String(), err)
	}
	return env.Error.Code
}

func TestCreateLabourAndPartPosition(t *testing.T) {
	pool, router := newItemsTestEnv(t)
	orderNumber := seedOrder(t, pool, model.StatusRequested)

	rec := doItemRequest(t, router, http.MethodPost, itemPath(orderNumber),
		`{"kind":"labour","description":"Bremsbeläge vorne","hours":2.5,"unit_price_cents":8900}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST labour = %d, want 201 (body %q)", rec.Code, rec.Body.String())
	}
	labour := decodeItemResponse(t, rec)
	if labour.Kind != model.ItemKindLabour {
		t.Errorf("labour kind = %q, want %q", labour.Kind, model.ItemKindLabour)
	}
	if labour.Hours != 2.5 {
		t.Errorf("labour hours = %v, want 2.5", labour.Hours)
	}
	if labour.UnitPriceCents != 8900 {
		t.Errorf("labour unit price = %d, want 8900", labour.UnitPriceCents)
	}
	if labour.LineTotalCents != 22250 {
		t.Errorf("labour line total = %d, want 22250", labour.LineTotalCents)
	}
	if labour.Quantity != 0 {
		t.Errorf("labour quantity = %d, want 0", labour.Quantity)
	}
	if labour.ID == 0 {
		t.Error("labour item has no id")
	}

	rec = doItemRequest(t, router, http.MethodPost, itemPath(orderNumber),
		`{"kind":"part","description":"Ölfilter","quantity":3,"unit_price_cents":1500}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST part = %d, want 201 (body %q)", rec.Code, rec.Body.String())
	}
	part := decodeItemResponse(t, rec)
	if part.Kind != model.ItemKindPart {
		t.Errorf("part kind = %q, want %q", part.Kind, model.ItemKindPart)
	}
	if part.Quantity != 3 {
		t.Errorf("part quantity = %d, want 3", part.Quantity)
	}
	if part.UnitPriceCents != 1500 {
		t.Errorf("part unit price = %d, want 1500", part.UnitPriceCents)
	}
	if part.LineTotalCents != 4500 {
		t.Errorf("part line total = %d, want 4500", part.LineTotalCents)
	}
	if part.Hours != 0 {
		t.Errorf("part hours = %v, want 0", part.Hours)
	}
}

// TestLabourSnapshotsConfiguredRate proves a labour position without an explicit
// unit price snapshots HOURLY_RATE_CENTS and computes its line total from it.
func TestLabourSnapshotsConfiguredRate(t *testing.T) {
	pool, router := newItemsTestEnv(t)
	t.Setenv("HOURLY_RATE_CENTS", "8900")
	orderNumber := seedOrder(t, pool, model.StatusRequested)

	labour := createItem(t, router, orderNumber,
		`{"kind":"labour","description":"Inspektion","hours":2}`)
	if labour.UnitPriceCents != 8900 {
		t.Errorf("snapshotted unit price = %d, want 8900", labour.UnitPriceCents)
	}
	if labour.LineTotalCents != 17800 {
		t.Errorf("line total = %d, want 17800", labour.LineTotalCents)
	}
}

func TestUpdateLabourAndPartPosition(t *testing.T) {
	pool, router := newItemsTestEnv(t)
	orderNumber := seedOrder(t, pool, model.StatusRequested)

	labour := createItem(t, router, orderNumber,
		`{"kind":"labour","description":"Bremsen","hours":2.5,"unit_price_cents":8900}`)

	rec := doItemRequest(t, router, http.MethodPatch, itemPathWithID(orderNumber, labour.ID),
		`{"hours":2}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("PATCH labour hours = %d, want 200 (body %q)", rec.Code, rec.Body.String())
	}
	updated := decodeItemResponse(t, rec)
	if updated.Hours != 2 {
		t.Errorf("hours = %v, want 2", updated.Hours)
	}
	if updated.UnitPriceCents != 8900 {
		t.Errorf("unit price after hours PATCH = %d, want the stored 8900", updated.UnitPriceCents)
	}
	if updated.LineTotalCents != 17800 {
		t.Errorf("line total = %d, want 17800", updated.LineTotalCents)
	}

	rec = doItemRequest(t, router, http.MethodPatch, itemPathWithID(orderNumber, labour.ID),
		`{"unit_price_cents":10000}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("PATCH labour price = %d, want 200 (body %q)", rec.Code, rec.Body.String())
	}
	updated = decodeItemResponse(t, rec)
	if updated.UnitPriceCents != 10000 {
		t.Errorf("unit price = %d, want 10000", updated.UnitPriceCents)
	}
	if updated.LineTotalCents != 20000 {
		t.Errorf("line total = %d, want 20000", updated.LineTotalCents)
	}

	part := createItem(t, router, orderNumber,
		`{"kind":"part","description":"Ölfilter","quantity":3,"unit_price_cents":1500}`)

	rec = doItemRequest(t, router, http.MethodPatch, itemPathWithID(orderNumber, part.ID),
		`{"quantity":5}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("PATCH part quantity = %d, want 200 (body %q)", rec.Code, rec.Body.String())
	}
	updated = decodeItemResponse(t, rec)
	if updated.Quantity != 5 {
		t.Errorf("quantity = %d, want 5", updated.Quantity)
	}
	if updated.LineTotalCents != 7500 {
		t.Errorf("line total = %d, want 7500", updated.LineTotalCents)
	}
}

func TestDeletePosition(t *testing.T) {
	pool, router := newItemsTestEnv(t)
	orderNumber := seedOrder(t, pool, model.StatusRequested)

	part := createItem(t, router, orderNumber,
		`{"kind":"part","description":"Ölfilter","quantity":1,"unit_price_cents":1500}`)

	rec := doItemRequest(t, router, http.MethodDelete, itemPathWithID(orderNumber, part.ID), "")
	if rec.Code != http.StatusNoContent {
		t.Fatalf("DELETE part = %d, want 204 (body %q)", rec.Code, rec.Body.String())
	}

	rec = doItemRequest(t, router, http.MethodDelete, itemPathWithID(orderNumber, part.ID), "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("second DELETE = %d, want 404", rec.Code)
	}
	if code := decodeErrorCode(t, rec); code != CodeNotFound {
		t.Errorf("error code = %q, want %q", code, CodeNotFound)
	}
}

func TestCollectedOrderIsReadOnly(t *testing.T) {
	pool, router := newItemsTestEnv(t)
	ctx := context.Background()
	orderNumber := seedOrder(t, pool, model.StatusRequested)

	labour := createItem(t, router, orderNumber,
		`{"kind":"labour","description":"Bremsen","hours":2.5,"unit_price_cents":8900}`)

	if _, err := pool.Exec(ctx,
		`UPDATE workshop_orders SET status = $1 WHERE order_number = $2`,
		model.StatusPickedUp, orderNumber,
	); err != nil {
		t.Fatalf("set status to abgeholt: %v", err)
	}

	create := doItemRequest(t, router, http.MethodPost, itemPath(orderNumber),
		`{"kind":"part","description":"Ölfilter","quantity":1,"unit_price_cents":1500}`)
	if create.Code != http.StatusConflict {
		t.Fatalf("POST on collected order = %d, want 409 (body %q)", create.Code, create.Body.String())
	}
	patch := doItemRequest(t, router, http.MethodPatch, itemPathWithID(orderNumber, labour.ID),
		`{"hours":9}`)
	if patch.Code != http.StatusConflict {
		t.Fatalf("PATCH on collected order = %d, want 409 (body %q)", patch.Code, patch.Body.String())
	}
	del := doItemRequest(t, router, http.MethodDelete, itemPathWithID(orderNumber, labour.ID), "")
	if del.Code != http.StatusConflict {
		t.Fatalf("DELETE on collected order = %d, want 409 (body %q)", del.Code, del.Body.String())
	}
	for _, rec := range []*httptest.ResponseRecorder{create, patch, del} {
		if code := decodeErrorCode(t, rec); code != CodeConflict {
			t.Errorf("error code = %q, want %q", code, CodeConflict)
		}
	}

	var hours float64
	if err := pool.QueryRow(ctx, `SELECT COALESCE(hours, 0) FROM order_items WHERE id = $1`, labour.ID).Scan(&hours); err != nil {
		t.Fatalf("item disappeared from a collected order: %v", err)
	}
	if hours != 2.5 {
		t.Errorf("hours after rejected PATCH = %v, want the unchanged 2.5", hours)
	}
}

func TestInvalidPositionInput(t *testing.T) {
	pool, router := newItemsTestEnv(t)
	ctx := context.Background()
	orderNumber := seedOrder(t, pool, model.StatusRequested)

	cases := []struct {
		name string
		body string
	}{
		{"part without unit price", `{"kind":"part","description":"Filter","quantity":2}`},
		{"part without quantity", `{"kind":"part","description":"Filter","unit_price_cents":100}`},
		{"part with zero quantity", `{"kind":"part","description":"Filter","quantity":0,"unit_price_cents":100}`},
		{"labour without hours", `{"kind":"labour","description":"Arbeit","unit_price_cents":8900}`},
		{"labour with negative hours", `{"kind":"labour","description":"Arbeit","hours":-1,"unit_price_cents":8900}`},
		{"non-integer cents", `{"kind":"part","description":"Filter","quantity":2,"unit_price_cents":100.5}`},
		{"missing description", `{"kind":"labour","hours":2,"unit_price_cents":8900}`},
		{"unknown kind", `{"kind":"material","description":"Etwas","quantity":1,"unit_price_cents":100}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := doItemRequest(t, router, http.MethodPost, itemPath(orderNumber), tc.body)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("POST %s = %d, want 400 (body %q)", tc.name, rec.Code, rec.Body.String())
			}
			if code := decodeErrorCode(t, rec); code != CodeValidationFailed {
				t.Errorf("error code = %q, want %q", code, CodeValidationFailed)
			}
		})
	}

	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM order_items WHERE order_id = (
		SELECT id FROM workshop_orders WHERE order_number = $1)`, orderNumber).Scan(&count); err != nil {
		t.Fatalf("count items: %v", err)
	}
	if count != 0 {
		t.Errorf("invalid requests created %d positions, want 0", count)
	}
}

func TestUnknownOrderAndPosition(t *testing.T) {
	pool, router := newItemsTestEnv(t)
	orderNumber := seedOrder(t, pool, model.StatusRequested)

	create := doItemRequest(t, router, http.MethodPost, itemPath("AUF-GIBT-ES-NICHT"),
		`{"kind":"part","description":"Filter","quantity":1,"unit_price_cents":100}`)
	if create.Code != http.StatusNotFound {
		t.Fatalf("POST on unknown order = %d, want 404", create.Code)
	}

	patch := doItemRequest(t, router, http.MethodPatch, itemPathWithID(orderNumber, 999999), `{"quantity":2}`)
	if patch.Code != http.StatusNotFound {
		t.Fatalf("PATCH unknown item = %d, want 404", patch.Code)
	}

	del := doItemRequest(t, router, http.MethodDelete, itemPathWithID(orderNumber, 999999), "")
	if del.Code != http.StatusNotFound {
		t.Fatalf("DELETE unknown item = %d, want 404", del.Code)
	}

	for _, rec := range []*httptest.ResponseRecorder{create, patch, del} {
		if code := decodeErrorCode(t, rec); code != CodeNotFound {
			t.Errorf("error code = %q, want %q", code, CodeNotFound)
		}
	}
}
