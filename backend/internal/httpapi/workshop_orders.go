package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"workshop-api/internal/model"
	"workshop-api/internal/store"
)

// validOrderStatuses is the closed set of order statuses the list filter accepts.
var validOrderStatuses = map[string]struct{}{
	model.StatusRequested:  {},
	model.StatusConfirmed:  {},
	model.StatusInProgress: {},
	model.StatusDone:       {},
	model.StatusPickedUp:   {},
}

// handleListOrders serves GET /api/workshop/orders. It filters by the optional
// status (validated against the five allowed values) and the optional plate
// substring (case-insensitive) and returns the matching order rows.
func handleListOrders(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status := strings.TrimSpace(r.URL.Query().Get("status"))
		if status != "" {
			if _, ok := validOrderStatuses[status]; !ok {
				writeError(w, http.StatusBadRequest, CodeValidationFailed, "invalid status filter")
				return
			}
		}

		plate := strings.TrimSpace(r.URL.Query().Get("plate"))

		orders, err := st.ListOrders(r.Context(), status, plate)
		if err != nil {
			writeError(w, http.StatusInternalServerError, CodeInternalError, "Internal Server Error")
			return
		}
		if orders == nil {
			orders = []model.OrderRow{}
		}
		writeJSON(w, http.StatusOK, map[string]any{"orders": orders})
	}
}

// handleOrderDetail serves GET /api/workshop/orders/{order_number}. It returns
// the full order with customer, vehicle, positions, status history and
// subtotal; an unknown number answers 404 not_found.
func handleOrderDetail(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orderNumber := strings.TrimSpace(r.PathValue("order_number"))
		if orderNumber == "" {
			writeError(w, http.StatusNotFound, CodeNotFound, "order not found")
			return
		}

		detail, err := st.OrderDetail(r.Context(), orderNumber)
		if errors.Is(err, store.ErrOrderNotFound) {
			writeError(w, http.StatusNotFound, CodeNotFound, "order not found")
			return
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, CodeInternalError, "Internal Server Error")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"order": detail})
	}
}
