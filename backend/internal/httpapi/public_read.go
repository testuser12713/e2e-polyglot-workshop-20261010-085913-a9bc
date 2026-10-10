package httpapi

import (
	"net/http"

	"workshop-api/internal/store"
)

// handlePublicOrderStatus serves GET /api/public/orders/{order_number}. The
// public read ticket implements the body.
func handlePublicOrderStatus(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotImplemented, CodeNotImplemented, "not implemented")
	}
}

// handlePublicOrderInvoice serves
// GET /api/public/orders/{order_number}/invoice. The public read ticket
// implements the body.
func handlePublicOrderInvoice(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotImplemented, CodeNotImplemented, "not implemented")
	}
}
