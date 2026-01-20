package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIntegration_FullRequestFlow(t *testing.T) {
	helper := SetupTestEnv()
	defer helper.RestoreEnv()

	// Create a mock backend server
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Backend-Header", "backend-value")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hello from backend"))
	}))
	defer backend.Close()

	// Configure proxy to use the test backend
	helper.SetEnv("PROXY_BACKEND", backend.URL)

	config, err := LoadTestConfig()
	assert.NoError(t, err)

	// Create proxy server with logging
	loggingHandler := loggingMiddleware(reverseProxy(config.Backend))

	// Create a test request to the proxy
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	// Execute the request
	loggingHandler.ServeHTTP(w, req)

	// Verify response
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "Hello from backend", w.Body.String())
	assert.Equal(t, "backend-value", w.Header().Get("X-Backend-Header"))

	// Verify logging occurred
	logs := helper.GetLogs()
	assert.Contains(t, logs, "GET")
	assert.Contains(t, logs, "/test")
	assert.Contains(t, logs, "200")
}

func TestIntegration_LoggingOutput(t *testing.T) {
	helper := SetupTestEnv()
	defer helper.RestoreEnv()

	// Create a simple backend
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer backend.Close()

	helper.SetEnv("PROXY_BACKEND", backend.URL)

	config, err := LoadTestConfig()
	assert.NoError(t, err)

	loggingHandler := loggingMiddleware(reverseProxy(config.Backend))

	// Make multiple requests to test logging
	requests := []struct {
		method string
		path   string
	}{
		{"GET", "/api/users"},
		{"POST", "/api/create"},
		{"PUT", "/api/update/123"},
	}

	for _, req := range requests {
		httpReq := httptest.NewRequest(req.method, req.path, nil)
		w := httptest.NewRecorder()
		loggingHandler.ServeHTTP(w, httpReq)

		logs := helper.GetLogs()
		assert.Contains(t, logs, req.method)
		assert.Contains(t, logs, req.path)

		helper.ClearLogs() // Clear for next request
	}
}

func TestIntegration_ConfigToServer(t *testing.T) {
	helper := SetupTestEnv()
	defer helper.RestoreEnv()

	// Test that config changes affect the proxy behavior
	helper.SetEnv("PROXY_PORT", "9999")
	helper.SetEnv("PROXY_BACKEND", "http://nonexistent-backend.com")

	config, err := LoadTestConfig()
	assert.NoError(t, err)

	assert.Equal(t, 9999, config.Port)
	assert.Equal(t, "http://nonexistent-backend.com", config.Backend)
}

func TestIntegration_ErrorHandling(t *testing.T) {
	helper := SetupTestEnv()
	defer helper.RestoreEnv()

	// Use a backend that doesn't exist to test error handling
	helper.SetEnv("PROXY_BACKEND", "http://127.0.0.1:0") // Closed port

	config, err := LoadTestConfig()
	assert.NoError(t, err)

	loggingHandler := loggingMiddleware(reverseProxy(config.Backend))

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	// This should result in a 502 Bad Gateway or connection error
	loggingHandler.ServeHTTP(w, req)

	// The response might be 502 or some error status
	// The important thing is that the request was processed and logged
	logs := helper.GetLogs()
	assert.Contains(t, logs, "GET")
	assert.Contains(t, logs, "/test")
}

func TestIntegration_MiddlewareStack(t *testing.T) {
	helper := SetupTestEnv()
	defer helper.RestoreEnv()

	// Create backend
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer backend.Close()

	helper.SetEnv("PROXY_BACKEND", backend.URL)

	config, err := LoadTestConfig()
	assert.NoError(t, err)

	// Test that middleware is properly applied
	handler := loggingMiddleware(reverseProxy(config.Backend))

	req := httptest.NewRequest("GET", "/middleware-test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "OK", w.Body.String())

	// Verify logging happened
	logs := helper.GetLogs()
	assert.Contains(t, logs, "GET")
	assert.Contains(t, logs, "/middleware-test")
	assert.Contains(t, logs, "200")
}
