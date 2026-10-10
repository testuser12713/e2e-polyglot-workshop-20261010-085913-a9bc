package httpapi

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"workshop-api/internal/auth"
	"workshop-api/internal/config"
	"workshop-api/internal/store"
)

// The workshop login route allows at most loginRateLimit attempts per client
// and per e-mail within loginRateWindow (AC-29).
const (
	loginRateLimit  = 10
	loginRateWindow = time.Minute
)

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

// handleLogin serves POST /api/workshop/login. It caps attempts per client IP
// and e-mail, verifies the password against the stored bcrypt hash and issues a
// time-limited HS256 token. Unknown e-mail and wrong password answer with the
// identical 401 so the response never reveals which one failed (AC-14, AC-29).
func handleLogin(st *store.Store, cfg config.Config) http.HandlerFunc {
	limiter := NewRateLimiter(loginRateLimit, loginRateWindow)
	return func(w http.ResponseWriter, r *http.Request) {
		var req loginRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, CodeValidationFailed, "invalid request body")
			return
		}
		req.Email = strings.ToLower(strings.TrimSpace(req.Email))

		if !limiter.AllowLogin(clientIP(r), req.Email) {
			writeError(w, http.StatusTooManyRequests, CodeRateLimited, "too many login attempts")
			return
		}

		emp, hash, err := st.FindEmployeeByEmail(r.Context(), req.Email)
		if err != nil {
			if errors.Is(err, store.ErrEmployeeNotFound) {
				writeUnauthorized(w)
				return
			}
			writeError(w, http.StatusInternalServerError, CodeInternalError, "Internal Server Error")
			return
		}

		if bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Password)) != nil {
			writeUnauthorized(w)
			return
		}

		token, expiresAt, err := auth.IssueToken(emp, cfg.AuthTokenSecret, time.Duration(cfg.AuthTokenTTLMinutes)*time.Minute)
		if err != nil {
			writeError(w, http.StatusInternalServerError, CodeInternalError, "Internal Server Error")
			return
		}
		writeJSON(w, http.StatusOK, loginResponse{Token: token, ExpiresAt: expiresAt})
	}
}

// writeUnauthorized writes the single 401 body used for both unknown e-mail and
// wrong password, so the two cannot be told apart (AC-29).
func writeUnauthorized(w http.ResponseWriter) {
	writeError(w, http.StatusUnauthorized, CodeUnauthorized, "invalid email or password")
}

// clientIP resolves the caller's IP address, preferring the first entry of a
// proxy-provided X-Forwarded-For header and falling back to the peer address.
func clientIP(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		if first := strings.TrimSpace(strings.Split(forwarded, ",")[0]); first != "" {
			return first
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
