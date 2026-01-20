package main

import (
	"net/http"
	"strconv"
	"sync"
	"time"
)

// TokenBucket represents a token bucket for rate limiting
type TokenBucket struct {
	tokens     float64   // current number of tokens
	capacity   float64   // maximum tokens
	refillRate float64   // tokens per second
	lastRefill time.Time
	mu         sync.Mutex
}

// NewTokenBucket creates a new token bucket
func NewTokenBucket(capacity float64, refillRate float64) *TokenBucket {
	return &TokenBucket{
		tokens:     capacity, // start full
		capacity:   capacity,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

// Allow checks if a request can be allowed and consumes a token
func (tb *TokenBucket) Allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.refill()

	if tb.tokens >= 1.0 {
		tb.tokens -= 1.0
		return true
	}

	return false
}

// refill adds tokens based on elapsed time
func (tb *TokenBucket) refill() {
	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()
	tokensToAdd := elapsed * tb.refillRate

	tb.tokens = min(tb.capacity, tb.tokens+tokensToAdd)
	tb.lastRefill = now
}

// Tokens returns the current number of tokens
func (tb *TokenBucket) Tokens() float64 {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.refill()
	return tb.tokens
}

// RateLimiter manages rate limiting for multiple clients
type RateLimiter struct {
	buckets   map[string]*TokenBucket
	mu        sync.RWMutex
	rpm       int     // requests per minute
	burstSize int     // maximum burst size
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(rpm, burstSize int) *RateLimiter {
	return &RateLimiter{
		buckets:   make(map[string]*TokenBucket),
		rpm:       rpm,
		burstSize: burstSize,
	}
}

// Allow checks if a request from the given key is allowed
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	bucket, exists := rl.buckets[key]
	if !exists {
		// Create new bucket for this key
		refillRate := float64(rl.rpm) / 60.0 // tokens per second
		bucket = NewTokenBucket(float64(rl.burstSize), refillRate)
		rl.buckets[key] = bucket
	}

	return bucket.Allow()
}

// GetRemainingTokens returns remaining tokens for a key
func (rl *RateLimiter) GetRemainingTokens(key string) int {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	if bucket, exists := rl.buckets[key]; exists {
		return int(bucket.Tokens())
	}

	return rl.burstSize // full bucket for new keys
}

// GetResetTime returns when the bucket will be fully refilled
func (rl *RateLimiter) GetResetTime(key string) time.Time {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	if bucket, exists := rl.buckets[key]; exists {
		remainingTokens := bucket.capacity - bucket.Tokens()
		if remainingTokens <= 0 {
			return time.Now()
		}

		secondsToRefill := remainingTokens / bucket.refillRate
		return time.Now().Add(time.Duration(secondsToRefill) * time.Second)
	}

	return time.Now()
}

// Cleanup removes old buckets to prevent memory leaks
func (rl *RateLimiter) Cleanup(maxAge time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	for key, bucket := range rl.buckets {
		// Remove buckets that haven't been used recently
		if bucket.lastRefill.Before(cutoff) && bucket.Tokens() >= bucket.capacity {
			delete(rl.buckets, key)
		}
	}
}

// Stats returns rate limiter statistics
func (rl *RateLimiter) Stats() (buckets int, totalTokens float64) {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	buckets = len(rl.buckets)
	for _, bucket := range rl.buckets {
		totalTokens += bucket.Tokens()
	}

	return buckets, totalTokens
}

// getClientKey extracts a client identifier from the request
func getClientKey(r *http.Request) string {
	// Use X-Forwarded-For if available, otherwise use RemoteAddr
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP in case of multiple
		if comma := len(xff); comma > 0 {
			for i, char := range xff {
				if char == ',' {
					return xff[:i]
				}
			}
		}
		return xff
	}

	// Extract IP from RemoteAddr (format: "IP:port")
	if remoteAddr := r.RemoteAddr; remoteAddr != "" {
		for i := len(remoteAddr) - 1; i >= 0; i-- {
			if remoteAddr[i] == ':' {
				return remoteAddr[:i]
			}
		}
		return remoteAddr
	}

	// Fallback to a default key
	return "unknown"
}

// rateLimitMiddleware provides rate limiting functionality
func rateLimitMiddleware(limiter *RateLimiter, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if limiter == nil {
			next.ServeHTTP(w, r)
			return
		}

		clientKey := getClientKey(r)

		if !limiter.Allow(clientKey) {
			// Rate limit exceeded
			resetTime := limiter.GetResetTime(clientKey)

			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(limiter.rpm))
			w.Header().Set("X-RateLimit-Remaining", "0")
			w.Header().Set("X-RateLimit-Reset", resetTime.Format(time.RFC3339))
			w.Header().Set("Retry-After", strconv.Itoa(int(resetTime.Sub(time.Now()).Seconds())))

			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error":"rate_limit_exceeded","message":"Too many requests"}`))
			return
		}

		// Add rate limit headers to successful requests
		remaining := limiter.GetRemainingTokens(clientKey)
		resetTime := limiter.GetResetTime(clientKey)

		w.Header().Set("X-RateLimit-Limit", strconv.Itoa(limiter.rpm))
		w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
		w.Header().Set("X-RateLimit-Reset", resetTime.Format(time.RFC3339))

		next.ServeHTTP(w, r)
	})
}

// min returns the minimum of two float64 values
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
