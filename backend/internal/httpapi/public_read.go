package httpapi

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"workshop-api/internal/model"
	"workshop-api/internal/store"
)

// neutralLookupMessage is the one message of the public lookups. An unknown
// order number, a wrong plate, an unfinished order and a missing invoice all
// answer this exact body so a caller can never tell whether an order exists
// (AC-12, AC-13). It never contains any customer data (AC-35).
const neutralLookupMessage = "order not found"

// publicOrderStatusResponse is the body of GET /api/public/orders/{order_number}.
type publicOrderStatusResponse struct {
	OrderNumber   string              `json:"order_number"`
	Status        string              `json:"status"`
	StatusHistory []model.StatusEvent `json:"status_history"`
}

// publicInvoiceResponse is the body of
// GET /api/public/orders/{order_number}/invoice.
type publicInvoiceResponse struct {
	Invoice publicInvoiceView `json:"invoice"`
}

// publicInvoiceView is the invoice as shown in the customer portal.
type publicInvoiceView struct {
	Items         []model.InvoiceItem `json:"items"`
	SubtotalCents int64               `json:"subtotal_cents"`
	TaxCents      int64               `json:"tax_cents"`
	TotalCents    int64               `json:"total_cents"`
	CreatedAt     time.Time           `json:"created_at"`
}

// writeNeutralLookupNotFound writes the shared neutral 404 body.
func writeNeutralLookupNotFound(w http.ResponseWriter) {
	writeError(w, http.StatusNotFound, CodeNotFound, neutralLookupMessage)
}

// handlePublicOrderStatus serves GET /api/public/orders/{order_number}?plate=.
// The order is identified by its number together with the normalized plate; any
// mismatch answers one neutral 404 (AC-12).
func handlePublicOrderStatus(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orderNumber := strings.TrimSpace(r.PathValue("order_number"))
		plate := strings.TrimSpace(r.URL.Query().Get("plate"))
		if orderNumber == "" || plate == "" {
			writeNeutralLookupNotFound(w)
			return
		}

		order, history, err := st.OrderByNumberAndPlate(r.Context(), orderNumber, store.NormalizePlate(plate))
		if err != nil {
			if errors.Is(err, store.ErrOrderLookupNotFound) {
				writeNeutralLookupNotFound(w)
				return
			}
			writeError(w, http.StatusInternalServerError, CodeInternalError, "Internal Server Error")
			return
		}

		writeJSON(w, http.StatusOK, publicOrderStatusResponse{
			OrderNumber:   order.OrderNumber,
			Status:        order.Status,
			StatusHistory: history,
		})
	}
}

// handlePublicOrderInvoice serves
// GET /api/public/orders/{order_number}/invoice?plate=. The invoice is only
// available for a finished order that matches the number and normalized plate;
// every other case answers the same neutral 404 (AC-13).
func handlePublicOrderInvoice(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orderNumber := strings.TrimSpace(r.PathValue("order_number"))
		plate := strings.TrimSpace(r.URL.Query().Get("plate"))
		if orderNumber == "" || plate == "" {
			writeNeutralLookupNotFound(w)
			return
		}

		invoice, err := st.InvoiceByOrderNumberAndPlate(r.Context(), orderNumber, store.NormalizePlate(plate))
		if err != nil {
			if errors.Is(err, store.ErrOrderLookupNotFound) {
				writeNeutralLookupNotFound(w)
				return
			}
			writeError(w, http.StatusInternalServerError, CodeInternalError, "Internal Server Error")
			return
		}

		writeJSON(w, http.StatusOK, publicInvoiceResponse{
			Invoice: publicInvoiceView{
				Items:         invoice.Items,
				SubtotalCents: invoice.SubtotalCents,
				TaxCents:      invoice.TaxCents,
				TotalCents:    invoice.TotalCents,
				CreatedAt:     invoice.CreatedAt,
			},
		})
	}
}
