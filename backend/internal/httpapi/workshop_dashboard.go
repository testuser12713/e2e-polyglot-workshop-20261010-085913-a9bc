package httpapi

import (
	"net/http"

	"workshop-api/internal/store"
)

// handleDashboard serves GET /api/workshop/dashboard. The dashboard ticket
// implements the body.
func handleDashboard(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotImplemented, CodeNotImplemented, "not implemented")
	}
}
