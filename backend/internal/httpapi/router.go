package httpapi

import (
	"net/http"

	"workshop-api/internal/config"
	"workshop-api/internal/store"
)

// NewRouter builds the API handler. Every contract route is mounted here; the
// /api/workshop/ group except the login route is wrapped with RequireAuth.
func NewRouter(cfg config.Config, st *store.Store) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/health", handleHealth(st))

	mux.HandleFunc("POST /api/public/appointments", handleCreateAppointment(st))
	mux.HandleFunc("GET /api/public/orders/{order_number}", handlePublicOrderStatus(st))
	mux.HandleFunc("GET /api/public/orders/{order_number}/invoice", handlePublicOrderInvoice(st))

	mux.HandleFunc("POST /api/workshop/login", handleLogin(st, cfg))

	mux.Handle("GET /api/workshop/orders", RequireAuth(handleListOrders(st)))
	mux.Handle("GET /api/workshop/orders/{order_number}", RequireAuth(handleOrderDetail(st)))
	mux.Handle("POST /api/workshop/orders/{order_number}/items", RequireAuth(handleAddItem(st)))
	mux.Handle("PATCH /api/workshop/orders/{order_number}/items/{item_id}", RequireAuth(handleUpdateItem(st)))
	mux.Handle("DELETE /api/workshop/orders/{order_number}/items/{item_id}", RequireAuth(handleDeleteItem(st)))
	mux.Handle("POST /api/workshop/orders/{order_number}/status", RequireAuth(handleSetStatus(st)))
	mux.Handle("GET /api/workshop/dashboard", RequireAuth(handleDashboard(st)))

	mux.HandleFunc("/", handleNotFound)

	var handler http.Handler = mux
	handler = Recover(handler)
	handler = CORS(cfg.FrontendOrigin)(handler)
	return handler
}

// handleNotFound answers every unknown route with 404 in the error envelope.
func handleNotFound(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotFound, CodeNotFound, "resource not found")
}
