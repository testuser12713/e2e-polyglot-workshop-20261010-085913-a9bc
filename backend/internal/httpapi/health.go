package httpapi

import (
	"context"
	"net/http"
	"time"

	"workshop-api/internal/store"
)

// handleHealth answers GET /api/health with {"status":"ok"} when the database
// is reachable, so health exercises the same configuration the routes use.
func handleHealth(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()
		if err := st.Ping(ctx); err != nil {
			writeError(w, http.StatusInternalServerError, CodeInternalError, "database is not reachable")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}
