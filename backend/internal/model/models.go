// Package model holds the domain types shared by the API, the stores and the
// HTTP handlers of the workshop portal.
package model

import "time"

// Order status values. The order lifecycle is
// angefragt -> bestaetigt -> in_arbeit -> fertig -> abgeholt.
const (
	StatusRequested  = "angefragt"
	StatusConfirmed  = "bestaetigt"
	StatusInProgress = "in_arbeit"
	StatusDone       = "fertig"
	StatusPickedUp   = "abgeholt"
)

// Item kind values for order and invoice positions.
const (
	ItemKindLabour = "labour"
	ItemKindPart   = "part"
)

// Customer is a person who requested workshop work.
type Customer struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Phone     string    `json:"phone"`
	CreatedAt time.Time `json:"created_at"`
}

// Vehicle is a customer's car, identified by its normalized plate.
type Vehicle struct {
	ID              int64     `json:"id"`
	Plate           string    `json:"plate"`
	PlateNormalized string    `json:"plate_normalized"`
	Brand           string    `json:"brand"`
	Model           string    `json:"model"`
	Mileage         int       `json:"mileage"`
	CreatedAt       time.Time `json:"created_at"`
}

// Order is a workshop job. PreferredDate is a date (no time component).
type Order struct {
	ID            int64     `json:"id"`
	OrderNumber   string    `json:"order_number"`
	CustomerID    int64     `json:"customer_id"`
	VehicleID     int64     `json:"vehicle_id"`
	Status        string    `json:"status"`
	PreferredDate time.Time `json:"preferred_date"`
	Description   string    `json:"description"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// OrderItem is a position of an order: labour hours or a part.
type OrderItem struct {
	ID             int64   `json:"id"`
	OrderID        int64   `json:"order_id"`
	Kind           string  `json:"kind"`
	Description    string  `json:"description"`
	Hours          float64 `json:"hours"`
	Quantity       int     `json:"quantity"`
	UnitPriceCents int64   `json:"unit_price_cents"`
	LineTotalCents int64   `json:"line_total_cents"`
}

// StatusEvent is one entry of an order's status history.
type StatusEvent struct {
	Status    string    `json:"status"`
	ChangedAt time.Time `json:"changed_at"`
}

// Invoice is the billed summary of a finished order. All amounts are whole cents.
type Invoice struct {
	ID            int64         `json:"id"`
	OrderNumber   string        `json:"order_number"`
	SubtotalCents int64         `json:"subtotal_cents"`
	TaxCents      int64         `json:"tax_cents"`
	TotalCents    int64         `json:"total_cents"`
	CreatedAt     time.Time     `json:"created_at"`
	Items         []InvoiceItem `json:"items"`
}

// InvoiceItem is one billed position.
type InvoiceItem struct {
	Kind           string  `json:"kind"`
	Description    string  `json:"description"`
	Hours          float64 `json:"hours"`
	Quantity       int     `json:"quantity"`
	UnitPriceCents int64   `json:"unit_price_cents"`
	LineTotalCents int64   `json:"line_total_cents"`
}

// Employee is a workshop staff member that can sign in.
type Employee struct {
	ID        int64     `json:"id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

// OrderDetail is the full order representation used by the workshop detail view.
type OrderDetail struct {
	OrderRow
	Customer      Customer      `json:"customer"`
	Vehicle       Vehicle       `json:"vehicle"`
	Description   string        `json:"description"`
	Items         []OrderItem   `json:"items"`
	StatusHistory []StatusEvent `json:"status_history"`
	SubtotalCents int64         `json:"subtotal_cents"`
}

// OrderRow is the compact order representation used by list and dashboard views.
type OrderRow struct {
	OrderNumber   string `json:"order_number"`
	Status        string `json:"status"`
	Plate         string `json:"plate"`
	PreferredDate string `json:"preferred_date"`
	CustomerName  string `json:"customer_name"`
	TotalCents    int64  `json:"total_cents"`
}
