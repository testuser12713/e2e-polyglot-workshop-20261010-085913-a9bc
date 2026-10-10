package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"workshop-api/internal/store"
)

// appointmentRequest is the JSON body of POST /api/public/appointments. It
// carries the three blocks of the public Terminanfrage (AC-11).
type appointmentRequest struct {
	Customer appointmentCustomer `json:"customer"`
	Vehicle  appointmentVehicle  `json:"vehicle"`
	Order    appointmentOrder    `json:"order"`
}

type appointmentCustomer struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Phone string `json:"phone"`
}

type appointmentVehicle struct {
	Plate   string `json:"plate"`
	Brand   string `json:"brand"`
	Model   string `json:"model"`
	Mileage int    `json:"mileage"`
}

type appointmentOrder struct {
	PreferredDate string `json:"preferred_date"`
	Description   string `json:"description"`
}

// appointmentResponse is the 201 body: only the generated order number and the
// starting status. It deliberately carries no customer data.
type appointmentResponse struct {
	OrderNumber string `json:"order_number"`
	Status      string `json:"status"`
}

// handleCreateAppointment serves POST /api/public/appointments: it validates
// the body (400 validation_failed), creates the customer, the vehicle and the
// order, and answers 201 with the order number (AC-11). A duplicate plate is a
// 409 conflict (AC-02). Neither the request nor any error ever echoes the
// customer's name, e-mail, phone or plate (AC-35).
func handleCreateAppointment(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

		var req appointmentRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, CodeValidationFailed, "invalid request body")
			return
		}

		if msg := validateAppointment(req); msg != "" {
			writeError(w, http.StatusBadRequest, CodeValidationFailed, msg)
			return
		}

		preferredDate, err := parsePreferredDate(req.Order.PreferredDate)
		if err != nil {
			writeError(w, http.StatusBadRequest, CodeValidationFailed, "order.preferred_date must be a valid date (YYYY-MM-DD)")
			return
		}

		ctx := r.Context()

		customer, err := st.UpsertCustomerByEmail(ctx,
			strings.TrimSpace(req.Customer.Name),
			strings.TrimSpace(req.Customer.Email),
			strings.TrimSpace(req.Customer.Phone),
		)
		if err != nil {
			writeError(w, http.StatusInternalServerError, CodeInternalError, "Internal Server Error")
			return
		}

		vehicle, err := st.CreateVehicle(ctx,
			strings.TrimSpace(req.Vehicle.Plate),
			strings.TrimSpace(req.Vehicle.Brand),
			strings.TrimSpace(req.Vehicle.Model),
			req.Vehicle.Mileage,
		)
		if errors.Is(err, store.ErrDuplicatePlate) {
			writeError(w, http.StatusConflict, CodeConflict, "a vehicle with this plate is already registered")
			return
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, CodeInternalError, "Internal Server Error")
			return
		}

		order, err := st.CreateOrder(ctx, customer.ID, vehicle.ID, preferredDate, strings.TrimSpace(req.Order.Description))
		if err != nil {
			writeError(w, http.StatusInternalServerError, CodeInternalError, "Internal Server Error")
			return
		}

		writeJSON(w, http.StatusCreated, appointmentResponse{
			OrderNumber: order.OrderNumber,
			Status:      order.Status,
		})
	}
}

// validateAppointment returns the first validation message for the required
// fields, or "" when the body is complete. The messages name the field, never
// its value, so an error response stays free of personal data (AC-35).
func validateAppointment(req appointmentRequest) string {
	switch {
	case strings.TrimSpace(req.Customer.Name) == "":
		return "customer.name is required"
	case strings.TrimSpace(req.Customer.Email) == "":
		return "customer.email is required"
	case !isValidEmail(req.Customer.Email):
		return "customer.email is not a valid address"
	case strings.TrimSpace(req.Vehicle.Plate) == "":
		return "vehicle.plate is required"
	case strings.TrimSpace(req.Order.PreferredDate) == "":
		return "order.preferred_date is required"
	case strings.TrimSpace(req.Order.Description) == "":
		return "order.description is required"
	}
	return ""
}

// isValidEmail accepts a plain RFC 5322 address (no display name).
func isValidEmail(raw string) bool {
	addr, err := mail.ParseAddress(strings.TrimSpace(raw))
	if err != nil {
		return false
	}
	return addr.Address == strings.TrimSpace(raw)
}

// parsePreferredDate accepts the HTML date input value (YYYY-MM-DD) and, as a
// fallback, a full RFC 3339 timestamp.
func parsePreferredDate(raw string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	for _, layout := range []string{"2006-01-02", time.RFC3339} {
		if t, err := time.Parse(layout, raw); err == nil {
			return t, nil
		}
	}
	return time.Time{}, errors.New("invalid date")
}
