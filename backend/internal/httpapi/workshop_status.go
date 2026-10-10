package httpapi

import (
	"net/http"

	"workshop-api/internal/store"
)

// handleSetStatus serves POST /api/workshop/orders/{order_number}/status. The
// status ticket implements the body.
func handleSetStatus(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotImplemented, CodeNotImplemented, "not implemented")
	}
}
