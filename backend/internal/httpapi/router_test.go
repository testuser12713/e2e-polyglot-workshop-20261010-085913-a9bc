package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"workshop-api/internal/config"
	"workshop-api/internal/store"
)

const testFrontendOrigin = "http://localhost:5173"

// newTestRouter builds the real router against the real PostgreSQL database
// from DATABASE_URL (no SQLite, no in-memory stand-in). The schema is created
// from the product's own migration file.
func newTestRouter(t *testing.T) http.Handler {
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

	return NewRouter(config.Config{FrontendOrigin: testFrontendOrigin}, store.New(pool))
}

func applyTestMigration(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	raw, err := os.ReadFile("../../migrations/0001_init.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	for _, stmt := range strings.Split(string(raw), ";") {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err := pool.Exec(ctx, stmt); err != nil {
			t.Fatalf("apply migration statement: %v", err)
		}
	}
}

type envelope struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func TestHealthReturnsOK(t *testing.T) {
	router := newTestRouter(t)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/health", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/health = %d, want 200 (body %q)", rec.Code, rec.Body.String())
	}
	var body struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Status != "ok" {
		t.Fatalf("status = %q, want ok", body.Status)
	}
}

func TestUnknownRouteReturnsNotFoundEnvelope(t *testing.T) {
	router := newTestRouter(t)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/nope", nil))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("GET /api/nope = %d, want 404", rec.Code)
	}
	var env envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode body %q: %v", rec.Body.String(), err)
	}
	if env.Error.Code != CodeNotFound {
		t.Fatalf("error code = %q, want %q", env.Error.Code, CodeNotFound)
	}
	if env.Error.Message == "" {
		t.Fatal("error message is empty")
	}
}

// TestStubRoutesAreWired asserts that the routes this skeleton mounts exist and
// are reachable (registered under the agreed path and verb). It deliberately
// does not assert a stub's temporary answer, because the feature tickets will
// replace those bodies.
func TestStubRoutesAreWired(t *testing.T) {
	router := newTestRouter(t)

	// A path that matches no registered route falls through to the catch-all
	// handler. Comparing against that response tells "registered, but answers
	// the designed neutral 404 for unknown input" (the public lookups do) apart
	// from "never wired at all".
	fallthroughRec := httptest.NewRecorder()
	router.ServeHTTP(fallthroughRec, httptest.NewRequest(http.MethodGet, "/api/definitely-not-a-route", nil))
	fallthroughBody := fallthroughRec.Body.String()

	// notFoundOK marks a single-resource lookup whose implemented handler
	// legitimately answers 404 for an unknown identifier; such a route is
	// covered by its own feature test rather than this wiring probe.
	cases := []struct {
		method     string
		path       string
		notFoundOK bool
	}{
		{http.MethodPost, "/api/public/appointments", false},
		{http.MethodGet, "/api/public/orders/ABC-1", true},
		{http.MethodGet, "/api/public/orders/ABC-1/invoice", true},
		{http.MethodPost, "/api/workshop/login", false},
		{http.MethodGet, "/api/workshop/orders", false},
		{http.MethodGet, "/api/workshop/orders/ABC-1", true},
		{http.MethodPost, "/api/workshop/orders/ABC-1/items", false},
		{http.MethodPatch, "/api/workshop/orders/ABC-1/items/1", false},
		{http.MethodDelete, "/api/workshop/orders/ABC-1/items/1", false},
		{http.MethodPost, "/api/workshop/orders/ABC-1/status", false},
		{http.MethodGet, "/api/workshop/dashboard", false},
	}
	for _, tc := range cases {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(tc.method, tc.path, nil))
		if rec.Code == http.StatusNotFound && !tc.notFoundOK {
			t.Errorf("%s %s returned 404 — route is not wired", tc.method, tc.path)
		}
		if rec.Code == http.StatusMethodNotAllowed {
			t.Errorf("%s %s returned 405 — wrong verb registered", tc.method, tc.path)
		}
		if rec.Code == http.StatusNotFound && rec.Body.String() == fallthroughBody {
			t.Errorf("%s %s fell through to the catch-all — route is not wired", tc.method, tc.path)
		}
	}
}

// TestCORSAllowsOnlyConfiguredOrigin proves the CORS middleware echoes exactly
// the configured origin and never a wildcard (AC-30).
func TestCORSAllowsOnlyConfiguredOrigin(t *testing.T) {
	router := newTestRouter(t)

	allowed := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	req.Header.Set("Origin", testFrontendOrigin)
	router.ServeHTTP(allowed, req)

	if got := allowed.Header().Get("Access-Control-Allow-Origin"); got != testFrontendOrigin {
		t.Fatalf("Access-Control-Allow-Origin = %q, want %q", got, testFrontendOrigin)
	}
	if got := allowed.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("Access-Control-Allow-Credentials = %q, want true", got)
	}

	denied := httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/health", nil)
	req.Header.Set("Origin", "http://evil.example")
	router.ServeHTTP(denied, req)

	if got := denied.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want empty for a foreign origin", got)
	}
}
