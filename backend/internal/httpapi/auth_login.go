package httpapi

import (
	"net/http"

	"workshop-api/internal/config"
	"workshop-api/internal/store"
)

// handleLogin serves POST /api/workshop/login. The login ticket implements the
// body, including the rate limit and the token issuing.
func handleLogin(st *store.Store, cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotImplemented, CodeNotImplemented, "not implemented")
	}
}
