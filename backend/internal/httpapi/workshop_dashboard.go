package httpapi

import (
	"net/http"

	"workshop-api/internal/model"
	"workshop-api/internal/store"
)

// dashboardResponse is the body of GET /api/workshop/dashboard. Open and
// FinishedToday are always present as arrays; RevenueMonthCents is a whole-cent
// integer (AC-21).
type dashboardResponse struct {
	Open              []model.OrderRow `json:"open"`
	FinishedToday     []model.OrderRow `json:"finished_today"`
	RevenueMonthCents int64            `json:"revenue_month_cents"`
}

// handleDashboard serves GET /api/workshop/dashboard. It reads the signed-in
// employee from the request context (set by RequireAuth) and returns the open
// orders, the orders finished today (UTC) and the month's revenue in whole
// cents (AC-21, AC-16).
func handleDashboard(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := EmployeeFromContext(r.Context()); !ok {
			writeError(w, http.StatusUnauthorized, CodeUnauthorized, "authentication required")
			return
		}

		open, finishedToday, revenueMonthCents, err := st.Dashboard(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, CodeInternalError, "Internal Server Error")
			return
		}
		if open == nil {
			open = []model.OrderRow{}
		}
		if finishedToday == nil {
			finishedToday = []model.OrderRow{}
		}

		writeJSON(w, http.StatusOK, dashboardResponse{
			Open:              open,
			FinishedToday:     finishedToday,
			RevenueMonthCents: revenueMonthCents,
		})
	}
}
