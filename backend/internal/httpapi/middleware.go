package httpapi

import (
	"context"
	"log"
	"net/http"

	"workshop-api/internal/model"
)

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

// RequireAuth guards the workshop routes. The skeleton passes the request
// through and sets no employee; the auth ticket fills in the bearer-token check.
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
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
