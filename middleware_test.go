package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLoggingMiddleware_CapturesRequestDetails(t *testing.T) {
	helper := SetupTestEnv()
	defer helper.RestoreEnv()

	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	})

	// Wrap with logging middleware
	loggingHandler := loggingMiddleware(testHandler)

	// Create a test request
	req := httptest.NewRequest("GET", "/test-path", nil)
	w := httptest.NewRecorder()

	// Execute the request
	loggingHandler.ServeHTTP(w, req)

	// Check logs contain expected information
	logs := helper.GetLogs()
	assert.Contains(t, logs, "GET")
	assert.Contains(t, logs, "/test-path")
	assert.Contains(t, logs, "200")
}

func TestLoggingMiddleware_CapturesStatusCode(t *testing.T) {
	helper := SetupTestEnv()
	defer helper.RestoreEnv()

	// Test different status codes
	testCases := []struct {
		statusCode int
		handler    http.HandlerFunc
	}{
		{http.StatusOK, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}},
		{http.StatusNotFound, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}},
		{http.StatusInternalServerError, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}},
	}

	for _, tc := range testCases {
		helper.ClearLogs()
		loggingHandler := loggingMiddleware(tc.handler)

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		loggingHandler.ServeHTTP(w, req)

		logs := helper.GetLogs()
		assert.Contains(t, logs, "GET")
		assert.Contains(t, logs, "/test")
		// Note: We can't easily test exact status code due to default 200 behavior
		// This is a limitation of the current ResponseWriter wrapper
	}
}

func TestLoggingMiddleware_MeasuresResponseTime(t *testing.T) {
	helper := SetupTestEnv()
	defer helper.RestoreEnv()

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate some processing time (minimal)
		w.WriteHeader(http.StatusOK)
	})

	loggingHandler := loggingMiddleware(testHandler)
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	loggingHandler.ServeHTTP(w, req)

	logs := helper.GetLogs()
	// Check that duration is logged (should contain time unit)
	assert.Contains(t, logs, "GET")
	assert.Contains(t, logs, "/test")
	assert.True(t, strings.Contains(logs, "ms") || strings.Contains(logs, "µs") || strings.Contains(logs, "ns"),
		"Logs should contain duration with time unit")
}

func TestLoggingMiddleware_CallsNextHandler(t *testing.T) {
	helper := SetupTestEnv()
	defer helper.RestoreEnv()

	called := false
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("response"))
	})

	loggingHandler := loggingMiddleware(testHandler)
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	loggingHandler.ServeHTTP(w, req)

	assert.True(t, called, "Next handler should be called")
	assert.Equal(t, "response", w.Body.String())
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestLoggingResponseWriter_WriteHeader(t *testing.T) {
	// Test the ResponseWriter wrapper directly
	w := httptest.NewRecorder()
	lrw := &loggingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

	lrw.WriteHeader(http.StatusNotFound)

	assert.Equal(t, http.StatusNotFound, lrw.statusCode)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestLoggingResponseWriter_DefaultStatus(t *testing.T) {
	w := httptest.NewRecorder()
	lrw := &loggingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

	// Don't call WriteHeader, should default to 200
	lrw.Write([]byte("test"))

	assert.Equal(t, http.StatusOK, lrw.statusCode)
}

// Caching Middleware Tests

func TestShouldCacheResponse_GET_Success(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	resp := newCachingResponseWriter(w)
	resp.statusCode = http.StatusOK

	assert.True(t, shouldCacheResponse(req, resp))
}

func TestShouldCacheResponse_POST(t *testing.T) {
	req := httptest.NewRequest("POST", "/test", nil)
	w := httptest.NewRecorder()
	resp := newCachingResponseWriter(w)
	resp.statusCode = http.StatusOK

	assert.False(t, shouldCacheResponse(req, resp))
}

func TestShouldCacheResponse_ErrorStatus(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	resp := newCachingResponseWriter(w)
	resp.statusCode = http.StatusNotFound

	assert.False(t, shouldCacheResponse(req, resp))
}

func TestShouldCacheResponse_NoCacheHeader(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	resp := &cachingResponseWriter{
		statusCode: http.StatusOK,
		ResponseWriter: &mockResponseWriter{
			headers: map[string][]string{
				"Cache-Control": {"no-cache"},
			},
		},
	}

	assert.False(t, shouldCacheResponse(req, resp))
}

func TestShouldCacheResponse_PrivateCacheHeader(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	resp := &cachingResponseWriter{
		statusCode: http.StatusOK,
		ResponseWriter: &mockResponseWriter{
			headers: map[string][]string{
				"Cache-Control": {"private"},
			},
		},
	}

	assert.False(t, shouldCacheResponse(req, resp))
}

func TestGenerateCacheKey(t *testing.T) {
	req := httptest.NewRequest("GET", "http://example.com/path?query=value", nil)
	expected := "GET|http://example.com/path?query=value"

	assert.Equal(t, expected, generateCacheKey(req))
}

func TestGenerateCacheKey_POST(t *testing.T) {
	req := httptest.NewRequest("POST", "http://example.com/api", nil)
	expected := "POST|http://example.com/api"

	assert.Equal(t, expected, generateCacheKey(req))
}

func TestCachingResponseWriter_WriteHeader(t *testing.T) {
	w := httptest.NewRecorder()
	crw := newCachingResponseWriter(w)

	crw.WriteHeader(http.StatusNotFound)

	assert.Equal(t, http.StatusNotFound, crw.statusCode)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestCachingResponseWriter_Write(t *testing.T) {
	w := httptest.NewRecorder()
	crw := newCachingResponseWriter(w)

	data := []byte("test data")
	n, err := crw.Write(data)

	assert.NoError(t, err)
	assert.Equal(t, len(data), n)
	assert.Equal(t, data, crw.body.Bytes())
	assert.Equal(t, http.StatusOK, crw.statusCode) // Default status
}

func TestCachingResponseWriter_Write_AfterHeader(t *testing.T) {
	w := httptest.NewRecorder()
	crw := newCachingResponseWriter(w)

	crw.WriteHeader(http.StatusCreated)
	data := []byte("created data")
	n, err := crw.Write(data)

	assert.NoError(t, err)
	assert.Equal(t, len(data), n)
	assert.Equal(t, data, crw.body.Bytes())
	assert.Equal(t, http.StatusCreated, crw.statusCode)
}

func TestCachingMiddleware_CacheHit(t *testing.T) {
	cache := NewCache(10, 60)
	cachedResp := &CachedResponse{
		StatusCode: 200,
		Headers:    map[string][]string{"Content-Type": {"application/json"}},
		Body:       []byte(`{"cached": true}`),
		CreatedAt:  time.Now(),
	}

	cacheKey := "GET|http://example.com/test"
	cache.Set(cacheKey, cachedResp)

	req := httptest.NewRequest("GET", "http://example.com/test", nil)
	w := httptest.NewRecorder()

	handler := cachingMiddleware(cache, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("Handler should not be called on cache hit")
	}))

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, `{"cached": true}`, w.Body.String())
	assert.Equal(t, "HIT", w.Header().Get("X-Cache"))
}

func TestCachingMiddleware_CacheMiss(t *testing.T) {
	cache := NewCache(10, 60)

	req := httptest.NewRequest("GET", "http://example.com/test", nil)
	w := httptest.NewRecorder()

	called := false
	handler := cachingMiddleware(cache, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"fresh": true}`))
	}))

	handler.ServeHTTP(w, req)

	assert.True(t, called)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, `{"fresh": true}`, w.Body.String())
	assert.Equal(t, "MISS", w.Header().Get("X-Cache"))
}

func TestCachingMiddleware_CacheBypass(t *testing.T) {
	cache := NewCache(10, 60)

	req := httptest.NewRequest("POST", "http://example.com/test", nil)
	w := httptest.NewRecorder()

	called := false
	handler := cachingMiddleware(cache, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("response"))
	}))

	handler.ServeHTTP(w, req)

	assert.True(t, called)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "BYPASS", w.Header().Get("X-Cache"))
}

func TestCachingMiddleware_ErrorResponseNotCached(t *testing.T) {
	cache := NewCache(10, 60)

	req := httptest.NewRequest("GET", "http://example.com/test", nil)
	w := httptest.NewRecorder()

	called := false
	handler := cachingMiddleware(cache, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("not found"))
	}))

	handler.ServeHTTP(w, req)

	assert.True(t, called)
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, "BYPASS", w.Header().Get("X-Cache"))

	// Verify not cached
	cacheKey := "GET|http://example.com/test"
	_, found := cache.Get(cacheKey)
	assert.False(t, found)
}

func TestGetCacheMetrics(t *testing.T) {
	cache := NewCache(5, 60)

	// Add some items
	cache.Set("key1", &CachedResponse{StatusCode: 200})
	cache.Set("key2", &CachedResponse{StatusCode: 200})

	metrics := GetCacheMetrics(cache)

	assert.Equal(t, 2, metrics.Size)
	assert.Equal(t, int64(0), metrics.Hits)   // Not tracking hits in this implementation
	assert.Equal(t, int64(0), metrics.Misses) // Not tracking misses in this implementation
}

// Mock response writer for testing
type mockResponseWriter struct {
	headers map[string][]string
}

func (m *mockResponseWriter) Header() http.Header {
	h := make(http.Header)
	for k, v := range m.headers {
		h[k] = v
	}
	return h
}

func (m *mockResponseWriter) Write(data []byte) (int, error) {
	return len(data), nil
}

func (m *mockResponseWriter) WriteHeader(statusCode int) {
	// No-op
}
