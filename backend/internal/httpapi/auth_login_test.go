package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"workshop-api/internal/auth"
	"workshop-api/internal/config"
	"workshop-api/internal/store"
)

const (
	loginTestEmail    = "chef@werkstatt.example"
	loginTestPassword = "correct horse battery staple"
	loginTestSecret   = "login-test-secret-please-change-me"
	loginTestTTL      = 60 * time.Minute
)

type loginEnvelope struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func loginTestConfig() config.Config {
	return config.Config{
		FrontendOrigin:           testFrontendOrigin,
		AuthTokenSecret:          loginTestSecret,
		AuthTokenTTLMinutes:      int(loginTestTTL / time.Minute),
		WorkshopEmployeeEmail:    loginTestEmail,
		WorkshopEmployeePassword: loginTestPassword,
	}
}

// newLoginTestServer builds the real router against the real PostgreSQL database
// from DATABASE_URL (no SQLite, no in-memory stand-in) with the workspace's own
// migration file, and seeds the configured employee the way startup does.
func newLoginTestServer(t *testing.T) (http.Handler, *store.Store) {
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
	applyLoginTestMigration(t, ctx, pool)

	st := store.New(pool)
	if _, err := pool.Exec(ctx, "DELETE FROM employees WHERE email = $1", loginTestEmail); err != nil {
		t.Fatalf("reset the test employee row: %v", err)
	}
	if err := st.EnsureFirstEmployee(ctx, loginTestEmail, loginTestPassword); err != nil {
		t.Fatalf("ensure first employee: %v", err)
	}

	return NewRouter(loginTestConfig(), st), st
}

func applyLoginTestMigration(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
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

func postLogin(t *testing.T, router http.Handler, email, password, remoteAddr string) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(map[string]string{"email": email, "password": password})
	if err != nil {
		t.Fatalf("marshal login body: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/workshop/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if remoteAddr != "" {
		req.RemoteAddr = remoteAddr
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func TestLoginWithCorrectCredentialsIssuesToken(t *testing.T) {
	router, _ := newLoginTestServer(t)

	rec := postLogin(t, router, loginTestEmail, loginTestPassword, "10.0.0.1:1234")
	if rec.Code != http.StatusOK {
		t.Fatalf("login = %d, want 200 (body %q)", rec.Code, rec.Body.String())
	}

	var body struct {
		Token     string    `json:"token"`
		ExpiresAt time.Time `json:"expires_at"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode login response %q: %v", rec.Body.String(), err)
	}
	if body.Token == "" {
		t.Fatal("login response carries no token")
	}
	if body.ExpiresAt.Before(time.Now()) {
		t.Fatalf("expires_at = %v is not in the future", body.ExpiresAt)
	}

	parsed, err := jwt.ParseWithClaims(body.Token, &auth.Claims{}, func(tok *jwt.Token) (any, error) {
		return []byte(loginTestSecret), nil
	})
	if err != nil {
		t.Fatalf("parse issued token: %v", err)
	}
	claims, ok := parsed.Claims.(*auth.Claims)
	if !ok || !parsed.Valid {
		t.Fatalf("issued token is not a valid auth.Claims token")
	}
	if claims.Email != loginTestEmail {
		t.Fatalf("email claim = %q, want %q", claims.Email, loginTestEmail)
	}
	if _, err := strconv.ParseInt(claims.Sub, 10, 64); err != nil {
		t.Fatalf("sub claim %q is not an employee id: %v", claims.Sub, err)
	}
}

func TestLoginUnknownEmailAndWrongPasswordAreIdentical401(t *testing.T) {
	router, _ := newLoginTestServer(t)

	wrongPassword := postLogin(t, router, loginTestEmail, "definitely-wrong", "10.0.0.2:1234")
	unknownEmail := postLogin(t, router, "nobody@example.com", loginTestPassword, "10.0.0.3:1234")

	if wrongPassword.Code != http.StatusUnauthorized {
		t.Fatalf("wrong password = %d, want 401", wrongPassword.Code)
	}
	if unknownEmail.Code != http.StatusUnauthorized {
		t.Fatalf("unknown e-mail = %d, want 401", unknownEmail.Code)
	}
	if wrongPassword.Body.String() != unknownEmail.Body.String() {
		t.Fatalf("responses differ: wrong password %q vs unknown e-mail %q",
			wrongPassword.Body.String(), unknownEmail.Body.String())
	}

	var env loginEnvelope
	if err := json.Unmarshal(wrongPassword.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode 401 body: %v", err)
	}
	if env.Error.Code != CodeUnauthorized {
		t.Fatalf("error code = %q, want %q", env.Error.Code, CodeUnauthorized)
	}
	if strings.Contains(strings.ToLower(wrongPassword.Body.String()), "token") {
		t.Fatal("401 body leaks a token")
	}
}

func TestLoginRateLimitsAfterTenAttempts(t *testing.T) {
	router, _ := newLoginTestServer(t)

	addr := "10.0.0.9:5555"
	for attempt := 1; attempt <= 10; attempt++ {
		rec := postLogin(t, router, loginTestEmail, "wrong-password", addr)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d = %d, want 401", attempt, rec.Code)
		}
	}

	rec := postLogin(t, router, loginTestEmail, "wrong-password", addr)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("11th attempt = %d, want 429", rec.Code)
	}
	var env loginEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode 429 body: %v", err)
	}
	if env.Error.Code != CodeRateLimited {
		t.Fatalf("error code = %q, want %q", env.Error.Code, CodeRateLimited)
	}
}

func TestEnsureFirstEmployeeDoesNotDuplicateOrOverwritePassword(t *testing.T) {
	_, st := newLoginTestServer(t)
	ctx := context.Background()

	if err := st.EnsureFirstEmployee(ctx, loginTestEmail, "a-different-password"); err != nil {
		t.Fatalf("second EnsureFirstEmployee: %v", err)
	}

	var count int
	if err := st.Pool().QueryRow(ctx, "SELECT count(*) FROM employees WHERE email = $1", loginTestEmail).Scan(&count); err != nil {
		t.Fatalf("count employees: %v", err)
	}
	if count != 1 {
		t.Fatalf("employees for %q = %d, want 1", loginTestEmail, count)
	}

	_, hash, err := st.FindEmployeeByEmail(ctx, loginTestEmail)
	if err != nil {
		t.Fatalf("FindEmployeeByEmail: %v", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(loginTestPassword)); err != nil {
		t.Fatal("stored password was overwritten by a second startup")
	}
}

func TestFindEmployeeByEmailReportsMissingEmployee(t *testing.T) {
	_, st := newLoginTestServer(t)
	if _, _, err := st.FindEmployeeByEmail(context.Background(), "missing@example.com"); err != store.ErrEmployeeNotFound {
		t.Fatalf("FindEmployeeByEmail(missing) = %v, want store.ErrEmployeeNotFound", err)
	}
}
