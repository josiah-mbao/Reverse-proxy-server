package routes

import (
	"net/http"
	"reverse-proxy/handlers" // Replace with your module name
)

func InitializeRoutes() *http.ServeMux {
	router := http.NewServeMux()

	// Proxy route
	router.HandleFunc("/api/", handlers.ProxyHandler("http://backend-service:8080"))

	// Health check route
	router.HandleFunc("/health", handlers.HealthHandler)

	// Apply middleware
	withLogging := handlers.LoggingMiddleware(router)

	return withLogging
}