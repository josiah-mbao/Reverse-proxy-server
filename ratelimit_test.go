package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewTokenBucket(t *testing.T) {
	capacity := 10.0
	refillRate := 1.0 // 1 token per second
	bucket := NewTokenBucket(capacity, refillRate)

	assert.NotNil(t, bucket)
	assert.Equal(t, capacity, bucket.capacity)
	assert.Equal(t, refillRate, bucket.refillRate)
	assert.Equal(t, capacity, bucket.tokens) // Should start full
}

func TestTokenBucket_Allow(t *testing.T) {
	bucket := NewTokenBucket(5.0, 1.0)

	// Should allow 5 requests immediately
	for i := 0; i < 5; i++ {
		assert.True(t, bucket.Allow(), "Should allow request %d", i+1)
	}

	// Should deny the 6th request
	assert.False(t, bucket.Allow(), "Should deny 6th request")
}

func TestTokenBucket_Refill(t *testing.T) {
	bucket := NewTokenBucket(10.0, 2.0) // 2 tokens per second

	// Use all tokens
	for i := 0; i < 10; i++ {
		bucket.Allow()
	}

	// Should have very few tokens left (due to floating point precision)
	tokensLeft := bucket.Tokens()
	assert.True(t, tokensLeft < 0.1, "Should have very few tokens left, got %f", tokensLeft)

	// Wait 1 second and check refill
	time.Sleep(1100 * time.Millisecond)
	tokens := bucket.Tokens()
	assert.True(t, tokens >= 2.0, "Should have at least 2 tokens after 1 second, got %f", tokens)
}

func TestTokenBucket_Tokens(t *testing.T) {
	bucket := NewTokenBucket(10.0, 1.0)

	// Initially full
	assert.Equal(t, 10.0, bucket.Tokens())

	// Use some tokens
	bucket.Allow()
	bucket.Allow()
	tokens := bucket.Tokens()
	assert.True(t, tokens >= 7.9 && tokens <= 8.1, "Should have approximately 8 tokens, got %f", tokens)
}

func TestNewRateLimiter(t *testing.T) {
	rl := NewRateLimiter(100, 20)

	assert.NotNil(t, rl)
	assert.Equal(t, 100, rl.rpm)
	assert.Equal(t, 20, rl.burstSize)
	assert.NotNil(t, rl.buckets)
}

func TestRateLimiter_Allow_NewClient(t *testing.T) {
	rl := NewRateLimiter(10, 5) // 10 RPM, burst of 5

	// First request from new client should be allowed
	assert.True(t, rl.Allow("client1"))

	// Check that bucket was created
	rl.mu.RLock()
	_, exists := rl.buckets["client1"]
	rl.mu.RUnlock()
	assert.True(t, exists)
}

func TestRateLimiter_RateLimitExceeded(t *testing.T) {
	rl := NewRateLimiter(10, 2) // Very low limits for testing

	client := "test-client"

	// Use all burst tokens
	assert.True(t, rl.Allow(client))
	assert.True(t, rl.Allow(client))

	// Next request should be denied
	assert.False(t, rl.Allow(client))
}

func TestRateLimiter_GetRemainingTokens(t *testing.T) {
	rl := NewRateLimiter(10, 5)

	client := "test-client"

	// Initially full
	assert.Equal(t, 5, rl.GetRemainingTokens(client))

	// Use some tokens
	rl.Allow(client)
	rl.Allow(client)
	assert.Equal(t, 3, rl.GetRemainingTokens(client))
}

func TestRateLimiter_GetResetTime(t *testing.T) {
	rl := NewRateLimiter(60, 10) // 1 token per second

	client := "test-client"

	// Use all tokens
	for i := 0; i < 10; i++ {
		rl.Allow(client)
	}

	resetTime := rl.GetResetTime(client)
	assert.True(t, resetTime.After(time.Now()), "Reset time should be in the future")
}

func TestRateLimiter_Cleanup(t *testing.T) {
	rl := NewRateLimiter(10, 5)

	// Add a bucket and simulate old last refill time
	rl.Allow("client1")

	// Manually set the last refill time to be old
	rl.mu.Lock()
	if bucket, exists := rl.buckets["client1"]; exists {
		bucket.lastRefill = time.Now().Add(-2 * time.Hour) // 2 hours ago
	}
	rl.mu.Unlock()

	// Verify bucket exists
	rl.mu.RLock()
	assert.Equal(t, 1, len(rl.buckets))
	rl.mu.RUnlock()

	// Cleanup with 1 hour old cutoff (should remove bucket)
	rl.Cleanup(-time.Hour)

	rl.mu.RLock()
	assert.Equal(t, 0, len(rl.buckets))
	rl.mu.RUnlock()
}

func TestRateLimiter_Stats(t *testing.T) {
	rl := NewRateLimiter(10, 5)

	rl.Allow("client1")
	rl.Allow("client2")

	buckets, tokens := rl.Stats()
	assert.Equal(t, 2, buckets)
	assert.True(t, tokens > 0, "Should have some tokens")
}

func TestGetClientKey_XForwardedFor(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Forwarded-For", "192.168.1.100")

	key := getClientKey(req)
	assert.Equal(t, "192.168.1.100", key)
}

func TestGetClientKey_XForwardedForMultiple(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Forwarded-For", "192.168.1.100, 10.0.0.1, 172.16.0.1")

	key := getClientKey(req)
	assert.Equal(t, "192.168.1.100", key) // Should take first IP
}

func TestGetClientKey_RemoteAddr(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.100:12345"

	key := getClientKey(req)
	assert.Equal(t, "192.168.1.100", key)
}

func TestGetClientKey_Fallback(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "" // Clear any default RemoteAddr set by httptest

	key := getClientKey(req)
	assert.Equal(t, "unknown", key)
}

func TestRateLimitMiddleware_Allow(t *testing.T) {
	rl := NewRateLimiter(100, 10)
	handler := rateLimitMiddleware(rl, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "OK", w.Body.String())

	// Check rate limit headers
	assert.Equal(t, "100", w.Header().Get("X-RateLimit-Limit"))
	assert.Equal(t, "9", w.Header().Get("X-RateLimit-Remaining")) // 10 - 1 = 9
	assert.NotEmpty(t, w.Header().Get("X-RateLimit-Reset"))
}

func TestRateLimitMiddleware_Block(t *testing.T) {
	rl := NewRateLimiter(1, 1) // Very restrictive

	handler := rateLimitMiddleware(rl, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "127.0.0.1:12345"

	// First request should succeed
	w1 := httptest.NewRecorder()
	handler.ServeHTTP(w1, req)
	assert.Equal(t, http.StatusOK, w1.Code)

	// Second request should be blocked
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req)
	assert.Equal(t, http.StatusTooManyRequests, w2.Code)

	// Check response body
	body := w2.Body.String()
	assert.Contains(t, body, "rate_limit_exceeded")

	// Check headers
	assert.Equal(t, "1", w2.Header().Get("X-RateLimit-Limit"))
	assert.Equal(t, "0", w2.Header().Get("X-RateLimit-Remaining"))
	assert.NotEmpty(t, w2.Header().Get("Retry-After"))
}

func TestRateLimitMiddleware_Disabled(t *testing.T) {
	handler := rateLimitMiddleware(nil, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "OK", w.Body.String())

	// Should not have rate limit headers when disabled
	assert.Empty(t, w.Header().Get("X-RateLimit-Limit"))
}

func TestMin(t *testing.T) {
	assert.Equal(t, 5.0, min(5.0, 10.0))
	assert.Equal(t, 3.0, min(10.0, 3.0))
	assert.Equal(t, 7.0, min(7.0, 7.0))
}

func TestRateLimitMiddleware_Headers(t *testing.T) {
	rl := NewRateLimiter(60, 10) // 1 per second

	handler := rateLimitMiddleware(rl, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Verify all required headers are present
	limit := w.Header().Get("X-RateLimit-Limit")
	remaining := w.Header().Get("X-RateLimit-Remaining")
	reset := w.Header().Get("X-RateLimit-Reset")

	assert.NotEmpty(t, limit)
	assert.NotEmpty(t, remaining)
	assert.NotEmpty(t, reset)

	// Verify reset time is valid RFC3339
	_, err := time.Parse(time.RFC3339, reset)
	assert.NoError(t, err)
}

func TestTokenBucket_Concurrency(t *testing.T) {
	bucket := NewTokenBucket(100.0, 10.0)

	// Test concurrent access
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 10; j++ {
				bucket.Allow()
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should have used approximately 100 tokens (allowing for floating point precision)
	tokensLeft := bucket.Tokens()
	assert.True(t, tokensLeft < 1.0, "Should have used most tokens, got %f", tokensLeft)
}
