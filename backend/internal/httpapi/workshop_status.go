package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"workshop-api/internal/model"
	"workshop-api/internal/queue"
	"workshop-api/internal/store"
)

// setStatusRequest is the body of POST /api/workshop/orders/{order_number}/status.
type setStatusRequest struct {
	Status string `json:"status"`
}

// setStatusResponse is the order status returned after a successful transition,
// including the full, append-only status history (AC-06).
type setStatusResponse struct {
	OrderNumber   string              `json:"order_number"`
	Status        string              `json:"status"`
	StatusHistory []model.StatusEvent `json:"status_history"`
}

// handleSetStatus serves POST /api/workshop/orders/{order_number}/status. It
// reads the signed-in employee from the request context (set by RequireAuth),
// applies the transition and publishes the invoice message exactly once when
// the order becomes 'fertig' (AC-05, AC-06, AC-07).
func handleSetStatus(st *store.Store, pub *queue.Publisher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := EmployeeFromContext(r.Context()); !ok {
			writeError(w, http.StatusUnauthorized, CodeUnauthorized, "authentication required")
			return
		}

		orderNumber := strings.TrimSpace(r.PathValue("order_number"))
		if orderNumber == "" {
			writeError(w, http.StatusBadRequest, CodeValidationFailed, "order number is required")
			return
		}

		var req setStatusRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, CodeValidationFailed, "invalid request body")
			return
		}
		newStatus := strings.TrimSpace(req.Status)
		if !isKnownStatus(newStatus) {
			writeError(w, http.StatusBadRequest, CodeValidationFailed, "unknown status")
			return
		}

		var publish store.StatusPublisher
		if pub != nil {
			publish = pub.PublishInvoice
		}

		order, history, err := st.TransitionOrderStatus(r.Context(), orderNumber, newStatus, publish)
		if err != nil {
			switch {
			case errors.Is(err, store.ErrOrderNotFound):
				writeError(w, http.StatusNotFound, CodeNotFound, "order not found")
			case errors.Is(err, store.ErrStatusConflict):
				writeError(w, http.StatusConflict, CodeConflict, "status transition not allowed")
			default:
				writeError(w, http.StatusInternalServerError, CodeInternalError, "Internal Server Error")
			}
			return
		}

		writeJSON(w, http.StatusOK, setStatusResponse{
			OrderNumber:   order.OrderNumber,
			Status:        order.Status,
			StatusHistory: history,
		})
	}
}

// isKnownStatus reports whether s is one of the five lifecycle statuses.
func isKnownStatus(s string) bool {
	switch s {
	case model.StatusRequested,
		model.StatusConfirmed,
		model.StatusInProgress,
		model.StatusDone,
		model.StatusPickedUp:
		return true
	default:
		return false
	}
}
