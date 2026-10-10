package httpapi

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"strings"

	"workshop-api/internal/auth"
	"workshop-api/internal/model"
)

const bearerPrefix = "Bearer "

type contextKey int

const employeeContextKey contextKey = iota

// WithEmployee stores the signed-in employee in the request context.
func WithEmployee(ctx context.Context, emp model.Employee) context.Context {
	return context.WithValue(ctx, employeeContextKey, emp)
}

// EmployeeFromContext returns the signed-in employee stored by WithEmployee.
func EmployeeFromContext(ctx context.Context) (model.Employee, bool) {
	emp, ok := ctx.Value(employeeContextKey).(model.Employee)
	return emp, ok
}

// RequireAuth guards the workshop routes with the bearer token. It reads the
// Authorization: Bearer header, verifies the HS256 token and stores the
// signed-in employee in the request context; any missing, malformed, forged or
// expired token is answered with 401 unauthorized in the error envelope.
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := strings.TrimSpace(r.Header.Get("Authorization"))
		if len(header) <= len(bearerPrefix) || !strings.EqualFold(header[:len(bearerPrefix)], bearerPrefix) {
			writeError(w, http.StatusUnauthorized, CodeUnauthorized, "unauthorized")
			return
		}

		claims, err := auth.VerifyToken(strings.TrimSpace(header[len(bearerPrefix):]))
		if err != nil {
			writeError(w, http.StatusUnauthorized, CodeUnauthorized, "unauthorized")
			return
		}

		employeeID, err := strconv.ParseInt(strings.TrimSpace(claims.Sub), 10, 64)
		if err != nil {
			writeError(w, http.StatusUnauthorized, CodeUnauthorized, "unauthorized")
			return
		}

		emp := model.Employee{ID: employeeID, Email: claims.Email}
		next.ServeHTTP(w, r.WithContext(WithEmployee(r.Context(), emp)))
	})
}

// CORS allows exactly the configured frontend origin. It never answers with a
// wildcard origin, so credentialed requests stay safe (AC-30).
func CORS(origin string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if reqOrigin := r.Header.Get("Origin"); reqOrigin != "" && reqOrigin == origin {
				h := w.Header()
				h.Set("Access-Control-Allow-Origin", origin)
				h.Add("Vary", "Origin")
				h.Set("Access-Control-Allow-Credentials", "true")
				h.Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
				h.Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			}
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// Recover turns a panic into the generic 500 error envelope. It never puts the
// panic value, SQL text or a stack trace into the response or the log (AC-32).
func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("internal error handling %s %s", r.Method, r.URL.Path)
				writeError(w, http.StatusInternalServerError, CodeInternalError, "Internal Server Error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}
