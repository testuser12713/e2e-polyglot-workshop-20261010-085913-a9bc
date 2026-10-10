package httpapi

import (
	"net/http"

	"workshop-api/internal/store"
)

// handleAddItem serves POST /api/workshop/orders/{order_number}/items. The order
// item ticket implements the body.
func handleAddItem(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotImplemented, CodeNotImplemented, "not implemented")
	}
}

// handleUpdateItem serves PATCH
// /api/workshop/orders/{order_number}/items/{item_id}. The order item ticket
// implements the body.
func handleUpdateItem(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotImplemented, CodeNotImplemented, "not implemented")
	}
}

// handleDeleteItem serves DELETE
// /api/workshop/orders/{order_number}/items/{item_id}. The order item ticket
// implements the body.
func handleDeleteItem(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotImplemented, CodeNotImplemented, "not implemented")
	}
}
