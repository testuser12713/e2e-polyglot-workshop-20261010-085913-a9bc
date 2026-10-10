package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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

// uniqueSeq makes every test's email and plate distinct so repeated local runs
// never collide on the unique indexes.
var uniqueSeq atomic.Int64

func newAppointmentAPI(t *testing.T) (http.Handler, *pgxpool.Pool) {
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

	return NewRouter(config.Config{FrontendOrigin: testFrontendOrigin}, store.New(pool)), pool
}

func uniqueEmail(prefix string) string {
	n := uniqueSeq.Add(1)
	return fmt.Sprintf("kunde-%s-%d-%d@example.com", prefix, time.Now().UnixNano(), n)
}

func uniquePlate() string {
	n := uniqueSeq.Add(1)
	return fmt.Sprintf("XX-%d", time.Now().UnixNano()%1_000_000_000+n)
}

// appointmentBody builds a complete, valid request body for a given email and
// plate, so a test can vary exactly one dimension.
func appointmentBody(email, plate string) map[string]any {
	return map[string]any{
		"customer": map[string]any{
			"name":  "Maria Muster",
			"email": email,
			"phone": "0170 0000000",
		},
		"vehicle": map[string]any{
			"plate":   plate,
			"brand":   "VW",
			"model":   "Golf",
			"mileage": 120000,
		},
		"order": map[string]any{
			"preferred_date": "2026-11-03",
			"description":    "Bremsen prüfen",
		},
	}
}

func postAppointment(t *testing.T, router http.Handler, body any) *httptest.ResponseRecorder {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/public/appointments", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

// cleanupAppointment removes exactly the rows this test created, so the shared
// database is left as it was found. Only the tables of this ticket are touched.
func cleanupAppointment(t *testing.T, pool *pgxpool.Pool, email, plate string) {
	t.Helper()
	t.Cleanup(func() {
		ctx := context.Background()
		_, _ = pool.Exec(ctx, `DELETE FROM workshop_orders WHERE customer_id IN (SELECT id FROM customers WHERE email = $1)`, email)
		_, _ = pool.Exec(ctx, `DELETE FROM vehicles WHERE plate_normalized = $1`, store.NormalizePlate(plate))
		_, _ = pool.Exec(ctx, `DELETE FROM customers WHERE email = $1`, email)
	})
}

func decodeAppointmentResponse(t *testing.T, rec *httptest.ResponseRecorder) appointmentResponse {
	t.Helper()
	var resp appointmentResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response %q: %v", rec.Body.String(), err)
	}
	return resp
}

// TestAppointmentCreatesOrder answers 201 with a generated order number in
// status "angefragt" and persists customer, vehicle and order (AC-03, AC-11).
func TestAppointmentCreatesOrder(t *testing.T) {
	router, pool := newAppointmentAPI(t)
	email := uniqueEmail("create")
	plate := uniquePlate()
	cleanupAppointment(t, pool, email, plate)

	rec := postAppointment(t, router, appointmentBody(email, plate))
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /api/public/appointments = %d, want 201 (body %q)", rec.Code, rec.Body.String())
	}
	resp := decodeAppointmentResponse(t, rec)
	if resp.OrderNumber == "" {
		t.Fatal("order_number is empty")
	}
	if resp.Status != model.StatusRequested {
		t.Fatalf("status = %q, want %q", resp.Status, model.StatusRequested)
	}

	ctx := context.Background()
	var (
		orders   int
		history  int
		vehicles int
	)
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM workshop_orders WHERE order_number = $1`, resp.OrderNumber).Scan(&orders); err != nil {
		t.Fatalf("count orders: %v", err)
	}
	if orders != 1 {
		t.Fatalf("orders with the returned number = %d, want 1", orders)
	}
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM order_status_history h
		JOIN workshop_orders o ON o.id = h.order_id
		WHERE o.order_number = $1 AND h.status = $2`, resp.OrderNumber, model.StatusRequested).Scan(&history); err != nil {
		t.Fatalf("count history: %v", err)
	}
	if history != 1 {
		t.Fatalf("status history entries = %d, want 1", history)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM vehicles WHERE plate_normalized = $1`, store.NormalizePlate(plate)).Scan(&vehicles); err != nil {
		t.Fatalf("count vehicles: %v", err)
	}
	if vehicles != 1 {
		t.Fatalf("vehicles with the normalized plate = %d, want 1", vehicles)
	}
}

// TestAppointmentRepeatEmailReusesCustomer sends the same e-mail twice with two
// different plates and proves both orders belong to the same customer row
// (AC-01).
func TestAppointmentRepeatEmailReusesCustomer(t *testing.T) {
	router, pool := newAppointmentAPI(t)
	email := uniqueEmail("repeat")
	plateA := uniquePlate()
	plateB := uniquePlate()
	cleanupAppointment(t, pool, email, plateA)

	first := postAppointment(t, router, appointmentBody(email, plateA))
	if first.Code != http.StatusCreated {
		t.Fatalf("first appointment = %d, want 201 (body %q)", first.Code, first.Body.String())
	}
	second := postAppointment(t, router, appointmentBody(email, plateB))
	if second.Code != http.StatusCreated {
		t.Fatalf("second appointment = %d, want 201 (body %q)", second.Code, second.Body.String())
	}

	t.Cleanup(func() {
		ctx := context.Background()
		_, _ = pool.Exec(ctx, `DELETE FROM vehicles WHERE plate_normalized = $1`, store.NormalizePlate(plateB))
	})

	firstResp := decodeAppointmentResponse(t, first)
	secondResp := decodeAppointmentResponse(t, second)

	ctx := context.Background()
	customerID := func(orderNumber string) int64 {
		var id int64
		if err := pool.QueryRow(ctx, `SELECT customer_id FROM workshop_orders WHERE order_number = $1`, orderNumber).Scan(&id); err != nil {
			t.Fatalf("select customer id for %s: %v", orderNumber, err)
		}
		return id
	}
	if id1, id2 := customerID(firstResp.OrderNumber), customerID(secondResp.OrderNumber); id1 != id2 {
		t.Fatalf("customer ids differ: %d and %d, want the same row", id1, id2)
	}
}

// TestAppointmentDuplicatePlateReturns409 proves a second vehicle with the same
// plate in a different casing and spacing is rejected with 409 conflict (AC-02),
// and that the error body carries no plate or e-mail (AC-35).
func TestAppointmentDuplicatePlateReturns409(t *testing.T) {
	router, pool := newAppointmentAPI(t)
	email1 := uniqueEmail("dup1")
	email2 := uniqueEmail("dup2")
	base := fmt.Sprintf("AB-%d", uniqueSeq.Add(1)+time.Now().UnixNano()%100000)
	spaced := strings.ToLower(strings.Replace(base, "-", " ", 1))
	cleanupAppointment(t, pool, email1, base)
	cleanupAppointment(t, pool, email2, spaced)

	first := postAppointment(t, router, appointmentBody(email1, base))
	if first.Code != http.StatusCreated {
		t.Fatalf("first appointment = %d, want 201 (body %q)", first.Code, first.Body.String())
	}

	second := postAppointment(t, router, appointmentBody(email2, spaced))
	if second.Code != http.StatusConflict {
		t.Fatalf("duplicate plate = %d, want 409 (body %q)", second.Code, second.Body.String())
	}
	var env envelope
	if err := json.Unmarshal(second.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode error body %q: %v", second.Body.String(), err)
	}
	if env.Error.Code != CodeConflict {
		t.Fatalf("error code = %q, want %q", env.Error.Code, CodeConflict)
	}

	body := strings.ToLower(second.Body.String())
	if strings.Contains(body, strings.ToLower(email2)) || strings.Contains(body, strings.ToLower(store.NormalizePlate(spaced))) {
		t.Fatalf("error body leaks personal data: %q", second.Body.String())
	}
}

// TestAppointmentMissingFieldReturns400 proves an incomplete body is a
// validation error with the stable envelope (AC-22).
func TestAppointmentMissingFieldReturns400(t *testing.T) {
	router, _ := newAppointmentAPI(t)

	body := appointmentBody(uniqueEmail("missing"), uniquePlate())
	customer := body["customer"].(map[string]any)
	delete(customer, "name")

	rec := postAppointment(t, router, body)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("missing field = %d, want 400 (body %q)", rec.Code, rec.Body.String())
	}
	var env envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode error body %q: %v", rec.Body.String(), err)
	}
	if env.Error.Code != CodeValidationFailed {
		t.Fatalf("error code = %q, want %q", env.Error.Code, CodeValidationFailed)
	}
}

// TestAppointmentInvalidDateReturns400 proves an unparseable Wunschtermin is a
// validation error rather than a server error.
func TestAppointmentInvalidDateReturns400(t *testing.T) {
	router, _ := newAppointmentAPI(t)

	body := appointmentBody(uniqueEmail("baddate"), uniquePlate())
	body["order"].(map[string]any)["preferred_date"] = "03.11.2026"

	rec := postAppointment(t, router, body)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid date = %d, want 400 (body %q)", rec.Code, rec.Body.String())
	}
}
