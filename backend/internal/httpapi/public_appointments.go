package httpapi

import (
	"net/http"

	"workshop-api/internal/store"
)

// handleCreateAppointment serves POST /api/public/appointments. The public
// appointment ticket implements the body.
func handleCreateAppointment(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotImplemented, CodeNotImplemented, "not implemented")
	}
}
