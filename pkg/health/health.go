// Package health provides health check handlers for the API server.
package health

import "net/http"

// Handler returns an http.HandlerFunc that responds with the service health status.
func Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}
}
