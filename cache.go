package main

import (
	"container/list"
	"sync"
	"time"
)

// CacheEntry represents a cached response
type CacheEntry struct {
	Key        string
	Response   *CachedResponse
	Expiry     time.Time
}

// CachedResponse holds the cached HTTP response data
type CachedResponse struct {
	StatusCode int
	Headers    map[string][]string
	Body       []byte
	CreatedAt  time.Time
}

// Cache implements an LRU cache with TTL
type Cache struct {
	capacity int
	ttl      time.Duration
	items    map[string]*list.Element
	lru      *list.List
	mutex    sync.RWMutex
}

// NewCache creates a new LRU cache
func NewCache(capacity int, ttlSeconds int) *Cache {
	return &Cache{
		capacity: capacity,
		ttl:      time.Duration(ttlSeconds) * time.Second,
		items:    make(map[string]*list.Element),
		lru:      list.New(),
	}
}

// Get retrieves a cached response if it exists and hasn't expired
func (c *Cache) Get(key string) (*CachedResponse, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if elem, exists := c.items[key]; exists {
		entry := elem.Value.(*CacheEntry)

		// Check if expired
		if time.Now().After(entry.Expiry) {
			// Remove expired entry
			c.mutex.RUnlock()
			c.mutex.Lock()
			c.removeElement(elem)
			c.mutex.Unlock()
			c.mutex.RLock()
			return nil, false
		}

		// Move to front (most recently used)
		c.lru.MoveToFront(elem)
		return entry.Response, true
	}

	return nil, false
}

// Set stores a response in the cache
func (c *Cache) Set(key string, response *CachedResponse) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Check if key already exists
	if elem, exists := c.items[key]; exists {
		// Update existing entry
		entry := elem.Value.(*CacheEntry)
		entry.Response = response
		entry.Expiry = time.Now().Add(c.ttl)
		c.lru.MoveToFront(elem)
		return
	}

	// Create new entry
	entry := &CacheEntry{
		Key:      key,
		Response: response,
		Expiry:   time.Now().Add(c.ttl),
	}

	elem := c.lru.PushFront(entry)
	c.items[key] = elem

	// Evict if over capacity
	if c.lru.Len() > c.capacity {
		c.evict()
	}
}

// evict removes the least recently used item
func (c *Cache) evict() {
	elem := c.lru.Back()
	if elem != nil {
		entry := elem.Value.(*CacheEntry)
		delete(c.items, entry.Key)
		c.lru.Remove(elem)
	}
}

// removeElement removes a specific element from cache
func (c *Cache) removeElement(elem *list.Element) {
	entry := elem.Value.(*CacheEntry)
	delete(c.items, entry.Key)
	c.lru.Remove(elem)
}

// Clear removes all items from the cache
func (c *Cache) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.items = make(map[string]*list.Element)
	c.lru = list.New()
}

// Size returns the current number of items in cache
func (c *Cache) Size() int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.lru.Len()
}

// Stats returns cache statistics
func (c *Cache) Stats() (size int, capacity int) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.lru.Len(), c.capacity
}
