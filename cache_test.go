package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewCache(t *testing.T) {
	cache := NewCache(10, 60)
	assert.NotNil(t, cache)
	assert.Equal(t, 10, cache.capacity)
	assert.Equal(t, time.Duration(60)*time.Second, cache.ttl)
	assert.NotNil(t, cache.items)
	assert.NotNil(t, cache.lru)
}

func TestCache_SetAndGet(t *testing.T) {
	cache := NewCache(10, 60)
	response := &CachedResponse{
		StatusCode: 200,
		Headers:    map[string][]string{"Content-Type": {"application/json"}},
		Body:       []byte(`{"test": "data"}`),
		CreatedAt:  time.Now(),
	}

	// Test cache miss
	result, found := cache.Get("test-key")
	assert.False(t, found)
	assert.Nil(t, result)

	// Set cache entry
	cache.Set("test-key", response)

	// Test cache hit
	result, found = cache.Get("test-key")
	assert.True(t, found)
	assert.NotNil(t, result)
	assert.Equal(t, 200, result.StatusCode)
	assert.Equal(t, []byte(`{"test": "data"}`), result.Body)
}

func TestCache_TTLExpiration(t *testing.T) {
	cache := NewCache(10, 1) // 1 second TTL

	response := &CachedResponse{
		StatusCode: 200,
		Headers:    map[string][]string{},
		Body:       []byte("test"),
		CreatedAt:  time.Now(),
	}

	cache.Set("test-key", response)

	// Should be available immediately
	result, found := cache.Get("test-key")
	assert.True(t, found)
	assert.NotNil(t, result)

	// Wait for expiration
	time.Sleep(1100 * time.Millisecond)

	// Should be expired
	result, found = cache.Get("test-key")
	assert.False(t, found)
	assert.Nil(t, result)
}

func TestCache_Eviction(t *testing.T) {
	cache := NewCache(2, 60) // Capacity of 2

	// Fill cache
	cache.Set("key1", &CachedResponse{StatusCode: 200, Body: []byte("data1")})
	cache.Set("key2", &CachedResponse{StatusCode: 200, Body: []byte("data2")})

	assert.Equal(t, 2, cache.Size())

	// Add third item, should evict oldest
	cache.Set("key3", &CachedResponse{StatusCode: 200, Body: []byte("data3")})

	assert.Equal(t, 2, cache.Size())

	// key1 should be evicted (oldest), key2 and key3 should remain
	_, found := cache.Get("key1")
	assert.False(t, found)

	_, found = cache.Get("key2")
	assert.True(t, found)

	_, found = cache.Get("key3")
	assert.True(t, found)
}

func TestCache_LRUBehavior(t *testing.T) {
	cache := NewCache(3, 60)

	// Add items
	cache.Set("key1", &CachedResponse{StatusCode: 200, Body: []byte("data1")})
	cache.Set("key2", &CachedResponse{StatusCode: 200, Body: []byte("data2")})
	cache.Set("key3", &CachedResponse{StatusCode: 200, Body: []byte("data3")})

	// Access key1 to make it most recently used
	cache.Get("key1")

	// Add fourth item, should evict key2 (least recently used)
	cache.Set("key4", &CachedResponse{StatusCode: 200, Body: []byte("data4")})

	// key2 should be evicted
	_, found := cache.Get("key2")
	assert.False(t, found)

	// Others should remain
	_, found = cache.Get("key1")
	assert.True(t, found)

	_, found = cache.Get("key3")
	assert.True(t, found)

	_, found = cache.Get("key4")
	assert.True(t, found)
}

func TestCache_UpdateExistingKey(t *testing.T) {
	cache := NewCache(10, 60)

	response1 := &CachedResponse{
		StatusCode: 200,
		Body:       []byte("original"),
	}
	response2 := &CachedResponse{
		StatusCode: 201,
		Body:       []byte("updated"),
	}

	// Set initial value
	cache.Set("test-key", response1)

	result, found := cache.Get("test-key")
	assert.True(t, found)
	assert.Equal(t, []byte("original"), result.Body)

	// Update with same key
	cache.Set("test-key", response2)

	result, found = cache.Get("test-key")
	assert.True(t, found)
	assert.Equal(t, []byte("updated"), result.Body)
	assert.Equal(t, 201, result.StatusCode)
}

func TestCache_Clear(t *testing.T) {
	cache := NewCache(10, 60)

	cache.Set("key1", &CachedResponse{StatusCode: 200, Body: []byte("data1")})
	cache.Set("key2", &CachedResponse{StatusCode: 200, Body: []byte("data2")})

	assert.Equal(t, 2, cache.Size())

	cache.Clear()

	assert.Equal(t, 0, cache.Size())

	_, found := cache.Get("key1")
	assert.False(t, found)

	_, found = cache.Get("key2")
	assert.False(t, found)
}

func TestCache_Stats(t *testing.T) {
	cache := NewCache(5, 60)

	cache.Set("key1", &CachedResponse{StatusCode: 200, Body: []byte("data1")})
	cache.Set("key2", &CachedResponse{StatusCode: 200, Body: []byte("data2")})

	size, capacity := cache.Stats()
	assert.Equal(t, 2, size)
	assert.Equal(t, 5, capacity)
}

func TestCache_ConcurrentAccess(t *testing.T) {
	cache := NewCache(100, 60)

	// Test concurrent reads and writes
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			key := "key" + string(rune(id+'0'))
			response := &CachedResponse{
				StatusCode: 200,
				Body:       []byte("data" + string(rune(id+'0'))),
			}

			// Write
			cache.Set(key, response)

			// Read
			result, found := cache.Get(key)
			assert.True(t, found)
			assert.Equal(t, response.Body, result.Body)

			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestCache_Size(t *testing.T) {
	cache := NewCache(10, 60)

	assert.Equal(t, 0, cache.Size())

	cache.Set("key1", &CachedResponse{StatusCode: 200, Body: []byte("data1")})
	assert.Equal(t, 1, cache.Size())

	cache.Set("key2", &CachedResponse{StatusCode: 200, Body: []byte("data2")})
	assert.Equal(t, 2, cache.Size())
}
