package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"workshop-api/internal/auth"
	"workshop-api/internal/model"
)

const (
	testAuthSecret    = "test-auth-token-secret-please-rotate"
	testEmployeeEmail = "chef@werkstatt.example"
)

// signTestToken builds an HS256 token with the same contract claims the login
// ticket issues (sub = employee id, email, exp).
func signTestToken(t *testing.T, secret string, employeeID int64, expiresAt time.Time) string {
	t.Helper()
	claims := auth.Claims{
		Sub:   strconv.FormatInt(employeeID, 10),
		Email: testEmployeeEmail,
		Exp:   expiresAt.Unix(),
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, &claims).SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign test token: %v", err)
	}
	return signed
}

// workshopPaths are every workshop endpoint the router wraps with RequireAuth.
var workshopPaths = []struct {
	method string
	path   string
}{
	{http.MethodGet, "/api/workshop/orders"},
	{http.MethodGet, "/api/workshop/orders/WS-1"},
	{http.MethodPost, "/api/workshop/orders/WS-1/items"},
	{http.MethodPatch, "/api/workshop/orders/WS-1/items/1"},
	{http.MethodDelete, "/api/workshop/orders/WS-1/items/1"},
	{http.MethodPost, "/api/workshop/orders/WS-1/status"},
	{http.MethodGet, "/api/workshop/dashboard"},
}

func assertUnauthorized(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 (body %q)", rec.Code, rec.Body.String())
	}
	var env envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode body %q: %v", rec.Body.String(), err)
	}
	if env.Error.Code != CodeUnauthorized {
		t.Fatalf("error code = %q, want %q", env.Error.Code, CodeUnauthorized)
	}
}

// TestWorkshopEndpointsRejectMissingToken proves that every guarded workshop
// endpoint answers 401 without an Authorization header (AC-16).
func TestWorkshopEndpointsRejectMissingToken(t *testing.T) {
	t.Setenv("AUTH_TOKEN_SECRET", testAuthSecret)
	router := newTestRouter(t)

	for _, tc := range workshopPaths {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(tc.method, tc.path, nil))
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("%s %s = %d, want 401 (body %q)", tc.method, tc.path, rec.Code, rec.Body.String())
		}
	}
}

// TestWorkshopEndpointsRejectMalformedHeader proves a malformed Authorization
// header is answered with 401.
func TestWorkshopEndpointsRejectMalformedHeader(t *testing.T) {
	t.Setenv("AUTH_TOKEN_SECRET", testAuthSecret)
	router := newTestRouter(t)

	for _, header := range []string{"Token abc", "Bearer", "Bearer   ", "abc.def.ghi", "Basic dXNlcjpwYXNz"} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/workshop/orders", nil)
		req.Header.Set("Authorization", header)
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("Authorization %q = %d, want 401 (body %q)", header, rec.Code, rec.Body.String())
		}
	}
}

// TestWorkshopEndpointsRejectForgedToken proves a token signed with the wrong
// secret is answered with 401.
func TestWorkshopEndpointsRejectForgedToken(t *testing.T) {
	t.Setenv("AUTH_TOKEN_SECRET", testAuthSecret)
	router := newTestRouter(t)

	forged := signTestToken(t, "a-completely-different-secret", 1, time.Now().Add(time.Hour))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/workshop/orders", nil)
	req.Header.Set("Authorization", "Bearer "+forged)
	router.ServeHTTP(rec, req)
	assertUnauthorized(t, rec)
}

// TestWorkshopEndpointsRejectExpiredToken proves an expired token is answered
// with 401.
func TestWorkshopEndpointsRejectExpiredToken(t *testing.T) {
	t.Setenv("AUTH_TOKEN_SECRET", testAuthSecret)
	router := newTestRouter(t)

	expired := signTestToken(t, testAuthSecret, 1, time.Now().Add(-time.Minute))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/workshop/orders", nil)
	req.Header.Set("Authorization", "Bearer "+expired)
	router.ServeHTTP(rec, req)
	assertUnauthorized(t, rec)
}

// TestValidTokenReachesHandler proves a correctly signed token passes the
// middleware and the request reaches the wrapped handler (it must no longer
// answer 401). It deliberately does not assert the handler's own body, which
// belongs to the feature tickets.
func TestValidTokenReachesHandler(t *testing.T) {
	t.Setenv("AUTH_TOKEN_SECRET", testAuthSecret)
	router := newTestRouter(t)

	valid := signTestToken(t, testAuthSecret, 1, time.Now().Add(time.Hour))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/workshop/orders", nil)
	req.Header.Set("Authorization", "Bearer "+valid)
	router.ServeHTTP(rec, req)

	if rec.Code == http.StatusUnauthorized {
		t.Fatalf("valid token was rejected with 401 (body %q)", rec.Body.String())
	}
	if rec.Code == http.StatusNotFound || rec.Code == http.StatusMethodNotAllowed {
		t.Fatalf("valid token did not reach the handler: status %d", rec.Code)
	}
}

// TestRequireAuthStoresEmployee proves the verified claims land in the request
// context as the signed-in employee.
func TestRequireAuthStoresEmployee(t *testing.T) {
	t.Setenv("AUTH_TOKEN_SECRET", testAuthSecret)

	var got model.Employee
	var found bool
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got, found = EmployeeFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	valid := signTestToken(t, testAuthSecret, 42, time.Now().Add(time.Hour))
	req := httptest.NewRequest(http.MethodGet, "/api/workshop/orders", nil)
	req.Header.Set("Authorization", "Bearer "+valid)
	rec := httptest.NewRecorder()
	RequireAuth(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", rec.Code, rec.Body.String())
	}
	if !found {
		t.Fatal("employee not found in request context")
	}
	if got.ID != 42 {
		t.Fatalf("employee id = %d, want 42", got.ID)
	}
	if got.Email != testEmployeeEmail {
		t.Fatalf("employee email = %q, want %q", got.Email, testEmployeeEmail)
	}
}
