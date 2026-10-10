package httpapi

import (
	"log"
	"net/http"
	"strings"

	"workshop-api/internal/config"
	"workshop-api/internal/queue"
	"workshop-api/internal/store"
)

// NewRouter builds the API handler. Every contract route is mounted here; the
// /api/workshop/ group except the login route is wrapped with RequireAuth.
func NewRouter(cfg config.Config, st *store.Store) http.Handler {
	mux := http.NewServeMux()

	// The status transition publishes the invoice message to the Valkey stream.
	// The connection is lazy; an unset or unresolved VALKEY_URL leaves the
	// publisher nil and a finished order reports an internal error instead of
	// silently skipping the publish.
	pub, err := newInvoicePublisher(cfg.ValkeyURL)
	if err != nil {
		// Never log the URL itself: it carries Valkey credentials (AC-31, AC-35).
		log.Printf("valkey publisher unavailable")
	}

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
	mux.Handle("POST /api/workshop/orders/{order_number}/status", RequireAuth(handleSetStatus(st, pub)))
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

// newInvoicePublisher builds the Valkey publisher from the configured URL. An
// absent URL or an unresolved service placeholder yields a nil publisher so the
// router still boots; PublishInvoice is never constructed for a bogus address.
func newInvoicePublisher(url string) (*queue.Publisher, error) {
	url = strings.TrimSpace(url)
	if url == "" || strings.Contains(url, "${") {
		return nil, nil
	}
	return queue.NewPublisher(url)
}
