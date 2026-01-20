package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

// cachingResponseWriter captures the response for caching
type cachingResponseWriter struct {
	http.ResponseWriter
	statusCode int
	body       *bytes.Buffer
	headers    map[string][]string
	written    bool
}

func newCachingResponseWriter(w http.ResponseWriter) *cachingResponseWriter {
	return &cachingResponseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
		body:           &bytes.Buffer{},
		headers:        make(map[string][]string),
		written:        false,
	}
}

func (crw *cachingResponseWriter) WriteHeader(code int) {
	if !crw.written {
		crw.statusCode = code
		crw.ResponseWriter.WriteHeader(code)
		crw.written = true
	}
}

func (crw *cachingResponseWriter) Write(data []byte) (int, error) {
	if !crw.written {
		crw.WriteHeader(crw.statusCode)
	}
	crw.body.Write(data)
	return crw.ResponseWriter.Write(data)
}

func (crw *cachingResponseWriter) Header() http.Header {
	return crw.ResponseWriter.Header()
}

// shouldCacheResponse determines if a response should be cached
func shouldCacheResponse(req *http.Request, resp *cachingResponseWriter) bool {
	// Only cache GET requests
	if req.Method != http.MethodGet {
		return false
	}

	// Don't cache error responses
	if resp.statusCode >= 400 {
		return false
	}

	// Check Cache-Control header
	cacheControl := resp.Header().Get("Cache-Control")
	if strings.Contains(cacheControl, "no-cache") || strings.Contains(cacheControl, "private") {
		return false
	}

	return true
}

// generateCacheKey creates a unique key for the request
func generateCacheKey(req *http.Request) string {
	return req.Method + "|" + req.URL.String()
}

// cachingMiddleware provides response caching
func cachingMiddleware(cache *Cache, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cacheKey := generateCacheKey(r)

		// Try to get from cache first
		if cachedResp, found := cache.Get(cacheKey); found {
			// Serve from cache
			for key, values := range cachedResp.Headers {
				for _, value := range values {
					w.Header().Add(key, value)
				}
			}
			w.Header().Set("X-Cache", "HIT")
			w.WriteHeader(cachedResp.StatusCode)
			w.Write(cachedResp.Body)
			return
		}

		// Not in cache, wrap response writer to capture response
		crw := newCachingResponseWriter(w)
		next.ServeHTTP(crw, r)

		// Cache the response if appropriate
		if cache != nil && shouldCacheResponse(r, crw) {
			cachedResp := &CachedResponse{
				StatusCode: crw.statusCode,
				Headers:    make(map[string][]string),
				Body:       crw.body.Bytes(),
				CreatedAt:  time.Now(),
			}

			// Copy headers
			for key, values := range crw.Header() {
				cachedResp.Headers[key] = make([]string, len(values))
				copy(cachedResp.Headers[key], values)
			}

			cache.Set(cacheKey, cachedResp)
			w.Header().Set("X-Cache", "MISS")
		} else {
			w.Header().Set("X-Cache", "BYPASS")
		}
	})
}

// CacheMetrics holds cache performance metrics
type CacheMetrics struct {
	Hits   int64
	Misses int64
	Size   int
}

// GetCacheMetrics returns current cache metrics
func GetCacheMetrics(cache *Cache) CacheMetrics {
	size, _ := cache.Stats()
	return CacheMetrics{
		Hits:   0, // Would need to be tracked separately
		Misses: 0, // Would need to be tracked separately
		Size:   size,
	}
}

// ErrorResponse represents a structured error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
	Code    int    `json:"code,omitempty"`
}

// errorHandlingMiddleware provides centralized error handling and recovery
func errorHandlingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Recover from panics
		defer func() {
			if err := recover(); err != nil {
				log.Printf("Panic recovered: %v", err)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintf(w, `{"error":"internal_server_error","message":"An unexpected error occurred"}`)
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// timeoutMiddleware adds timeout handling to requests
func timeoutMiddleware(timeout time.Duration, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()

		r = r.WithContext(ctx)

		done := make(chan bool, 1)

		go func() {
			next.ServeHTTP(w, r)
			done <- true
		}()

		select {
		case <-done:
			// Request completed normally
			return
		case <-ctx.Done():
			// Timeout occurred
			if ctx.Err() == context.DeadlineExceeded {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusGatewayTimeout)
				fmt.Fprintf(w, `{"error":"gateway_timeout","message":"Request timed out"}`)
			}
		}
	})
}
