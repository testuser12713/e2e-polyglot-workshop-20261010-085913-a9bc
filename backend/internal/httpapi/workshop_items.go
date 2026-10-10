package httpapi

import (
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"

	"workshop-api/internal/model"
	"workshop-api/internal/store"
)

// itemRequest is the body of POST/PATCH item requests. Numbers are decoded as
// json.Number so a non-integer amount of cents (for example 1500.5) is a
// validation error rather than being silently truncated. An absent field is the
// empty string, which lets PATCH keep the stored value.
type itemRequest struct {
	Kind           string      `json:"kind"`
	Description    string      `json:"description"`
	Hours          json.Number `json:"hours"`
	UnitPriceCents json.Number `json:"unit_price_cents"`
	Quantity       json.Number `json:"quantity"`
}

// handleAddItem serves POST /api/workshop/orders/{order_number}/items. A labour
// position carries hours and an optional unit price snapshot (the configured
// hourly rate when omitted); a part position carries a quantity and a required
// unit price. All money is whole cents.
func handleAddItem(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !employeePresent(r) {
			writeError(w, http.StatusUnauthorized, CodeUnauthorized, "authentication required")
			return
		}

		req, ok := decodeItemRequest(w, r)
		if !ok {
			return
		}

		item, message := buildItem(req, model.OrderItem{}, false)
		if message != "" {
			writeError(w, http.StatusBadRequest, CodeValidationFailed, message)
			return
		}

		orderNumber := r.PathValue("order_number")
		created, err := st.AddOrderItem(r.Context(), orderNumber, item)
		if err != nil {
			writeItemStoreError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"item": created})
	}
}

// handleUpdateItem serves PATCH
// /api/workshop/orders/{order_number}/items/{item_id}. A partial body keeps the
// stored values of any field it omits. A collected order is read-only.
func handleUpdateItem(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !employeePresent(r) {
			writeError(w, http.StatusUnauthorized, CodeUnauthorized, "authentication required")
			return
		}

		orderNumber := r.PathValue("order_number")
		itemID, err := parseItemID(r.PathValue("item_id"))
		if err != nil {
			writeError(w, http.StatusNotFound, CodeNotFound, "position not found")
			return
		}

		existing, err := st.OrderItemByID(r.Context(), orderNumber, itemID)
		if err != nil {
			writeItemStoreError(w, err)
			return
		}

		req, ok := decodeItemRequest(w, r)
		if !ok {
			return
		}

		item, message := buildItem(req, existing, true)
		if message != "" {
			writeError(w, http.StatusBadRequest, CodeValidationFailed, message)
			return
		}

		updated, err := st.UpdateOrderItem(r.Context(), orderNumber, itemID, item)
		if err != nil {
			writeItemStoreError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"item": updated})
	}
}

// handleDeleteItem serves DELETE
// /api/workshop/orders/{order_number}/items/{item_id}. A collected order is
// read-only.
func handleDeleteItem(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !employeePresent(r) {
			writeError(w, http.StatusUnauthorized, CodeUnauthorized, "authentication required")
			return
		}

		orderNumber := r.PathValue("order_number")
		itemID, err := parseItemID(r.PathValue("item_id"))
		if err != nil {
			writeError(w, http.StatusNotFound, CodeNotFound, "position not found")
			return
		}

		if err := st.DeleteOrderItem(r.Context(), orderNumber, itemID); err != nil {
			writeItemStoreError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// employeePresent reports whether the guard placed the signed-in employee in the
// request context (via WithEmployee).
func employeePresent(r *http.Request) bool {
	_, ok := EmployeeFromContext(r.Context())
	return ok
}

// decodeItemRequest decodes the JSON body. A malformed body answers 400 and the
// function returns ok=false.
func decodeItemRequest(w http.ResponseWriter, r *http.Request) (itemRequest, bool) {
	var req itemRequest
	dec := json.NewDecoder(r.Body)
	dec.UseNumber()
	if err := dec.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, CodeValidationFailed, "invalid request body")
		return itemRequest{}, false
	}
	return req, true
}

// buildItem validates a create (isPatch=false) or merges a partial body onto an
// existing position (isPatch=true) and computes the line total. The returned
// message is empty on success, otherwise it is the validation error.
func buildItem(req itemRequest, existing model.OrderItem, isPatch bool) (model.OrderItem, string) {
	item := existing

	if req.Kind != "" {
		item.Kind = req.Kind
	}
	if item.Kind != model.ItemKindLabour && item.Kind != model.ItemKindPart {
		return item, `kind must be "labour" or "part"`
	}

	if req.Description != "" || !isPatch {
		item.Description = strings.TrimSpace(req.Description)
	}
	if item.Description == "" {
		return item, "description is required"
	}

	switch item.Kind {
	case model.ItemKindLabour:
		if req.Hours != "" {
			hours, err := strconv.ParseFloat(req.Hours.String(), 64)
			if err != nil {
				return item, "hours must be a number"
			}
			item.Hours = hours
		}
		if item.Hours <= 0 {
			return item, "hours must be greater than zero"
		}

		if req.UnitPriceCents != "" {
			cents, ok := parseIntegerCents(req.UnitPriceCents)
			if !ok {
				return item, "unit_price_cents must be an integer number of cents"
			}
			item.UnitPriceCents = cents
		} else if !isPatch {
			item.UnitPriceCents = configuredHourlyRateCents()
		}
		if item.UnitPriceCents < 0 {
			return item, "unit_price_cents must not be negative"
		}

		item.Quantity = 0
		item.LineTotalCents = int64(math.Round(item.Hours * float64(item.UnitPriceCents)))

	case model.ItemKindPart:
		if req.Quantity != "" {
			quantity, ok := parseIntegerCents(req.Quantity)
			if !ok {
				return item, "quantity must be a whole number"
			}
			item.Quantity = int(quantity)
		}
		if item.Quantity <= 0 {
			return item, "quantity must be greater than zero"
		}

		if req.UnitPriceCents != "" {
			cents, ok := parseIntegerCents(req.UnitPriceCents)
			if !ok {
				return item, "unit_price_cents must be an integer number of cents"
			}
			item.UnitPriceCents = cents
		} else if !isPatch {
			return item, "unit_price_cents is required for a part"
		}
		if item.UnitPriceCents < 0 {
			return item, "unit_price_cents must not be negative"
		}

		item.Hours = 0
		item.LineTotalCents = int64(item.Quantity) * item.UnitPriceCents
	}

	return item, ""
}

// parseIntegerCents parses a json.Number that must be a whole number of cents.
func parseIntegerCents(n json.Number) (int64, bool) {
	value, err := strconv.ParseInt(n.String(), 10, 64)
	if err != nil {
		return 0, false
	}
	return value, true
}

// parseItemID parses the {item_id} path segment.
func parseItemID(raw string) (int64, error) {
	return strconv.ParseInt(raw, 10, 64)
}

// configuredHourlyRateCents reads HOURLY_RATE_CENTS lazily and snapshots it onto
// a new labour position. An unset or unparsable value yields 0, which is a valid
// snapshot (the worker bills the authoritative rate at invoice time).
func configuredHourlyRateCents() int64 {
	raw := strings.TrimSpace(os.Getenv("HOURLY_RATE_CENTS"))
	if raw == "" {
		return 0
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0
	}
	return value
}

// writeItemStoreError maps a store error onto the shared error envelope.
func writeItemStoreError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrOrderNotFound), errors.Is(err, store.ErrItemNotFound):
		writeError(w, http.StatusNotFound, CodeNotFound, "order or position not found")
	case errors.Is(err, store.ErrOrderLocked):
		writeError(w, http.StatusConflict, CodeConflict, "the order is already collected")
	default:
		writeError(w, http.StatusInternalServerError, CodeInternalError, "Internal Server Error")
	}
}
