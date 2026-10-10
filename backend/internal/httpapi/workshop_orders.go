package httpapi

import (
	"net/http"

	"workshop-api/internal/store"
)

// handleListOrders serves GET /api/workshop/orders. The order list ticket
// implements the body.
func handleListOrders(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotImplemented, CodeNotImplemented, "not implemented")
	}
}

// handleOrderDetail serves GET /api/workshop/orders/{order_number}. The order
// list ticket implements the body.
func handleOrderDetail(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotImplemented, CodeNotImplemented, "not implemented")
	}
}
