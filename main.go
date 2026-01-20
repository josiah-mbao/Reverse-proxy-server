package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// loggingResponseWriter wraps http.ResponseWriter to capture status code
type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

// loggingMiddleware logs HTTP requests and responses
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap the ResponseWriter to capture status code
		lrw := &loggingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Call the next handler
		next.ServeHTTP(lrw, r)

		// Log the request
		duration := time.Since(start)
		log.Printf("%s %s %d %v", r.Method, r.URL.Path, lrw.statusCode, duration)
	})
}

func reverseProxy(target string) http.Handler {
	targetURL, err := url.Parse(target)
	if err != nil {
		fmt.Println("Error parsing URL: ", err)
		return nil
	}
	return httputil.NewSingleHostReverseProxy(targetURL)
}

func main() {
	config, err := LoadConfig()
	if err != nil {
		fmt.Printf("Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Starting proxy server on port %d, backend: %s\n", config.Port, config.Backend)

	// Create cache if enabled
	var cache *Cache
	if config.CacheEnabled {
		cache = NewCache(config.CacheSize, config.CacheTTL)
		fmt.Printf("Cache enabled: size=%d, ttl=%ds\n", config.CacheSize, config.CacheTTL)
	}

	// Create rate limiter if enabled
	var rateLimiter *RateLimiter
	if config.RateLimitEnabled {
		rateLimiter = NewRateLimiter(config.RateLimitRPM, config.RateLimitBurst)
		fmt.Printf("Rate limiting enabled: %d RPM, burst=%d\n", config.RateLimitRPM, config.RateLimitBurst)
	}

	// Build middleware chain
	handler := reverseProxy(config.Backend)
	handler = loggingMiddleware(handler)
	if cache != nil {
		handler = cachingMiddleware(cache, handler)
	}
	if rateLimiter != nil {
		handler = rateLimitMiddleware(rateLimiter, handler)
	}

	// Add error handling and timeout middleware
	handler = errorHandlingMiddleware(handler)
	handler = timeoutMiddleware(time.Duration(config.RequestTimeout)*time.Second, handler)

	// Create HTTP server
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", config.Port),
		Handler: handler,
	}

	// Channel to listen for interrupt signal
	done := make(chan bool, 1)
	quit := make(chan os.Signal, 1)

	// Register interrupt signals
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Start server in a goroutine
	go func() {
		log.Printf("Server starting on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Could not start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	<-quit
	log.Println("Server is shutting down...")

	// Create shutdown context with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Duration(config.ShutdownTimeout)*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	} else {
		log.Println("Server shutdown complete")
	}

	close(done)
	<-done
}
