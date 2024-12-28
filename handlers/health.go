package handlers

import (
	"net/http"
)

// HealthHandler responds with the health status.
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}