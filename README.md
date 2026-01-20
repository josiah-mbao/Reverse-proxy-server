# Building a Production-Ready HTTP Proxy Server in Go

A learning project documenting my journey from basic HTTP proxy to production-ready server architecture. This README captures the architectural decisions, technical challenges, and lessons learned while building enterprise-grade server infrastructure.

## 🏗️ Architecture Evolution

### Phase 1: Core HTTP Proxy (Foundation)
**Started with:** Basic reverse proxy using Go's `httputil.ReverseProxy`
**Challenge:** Understanding HTTP request lifecycle and middleware patterns
**Learning:** Go's standard library provides excellent HTTP primitives, but production needs require careful design

### Phase 2: Production Features (Reliability & Performance)
**Caching Layer:** LRU cache with TTL to reduce backend load
**Error Handling:** Panic recovery, structured error responses, timeout management
**Graceful Shutdown:** Signal handling, connection draining, resource cleanup
**Key Decision:** Middleware chain architecture allows composable features

### Technical Stack
- **Language:** Go 1.23+ (goroutines, channels, context)
- **HTTP:** Standard library with custom middleware
- **Caching:** Thread-safe LRU with TTL
- **Configuration:** Multi-source (env, file, flags) with precedence
- **Testing:** Comprehensive unit + integration tests

## 🏛️ Key Architectural Decisions

### Middleware Chain Design
**Decision:** Adopted middleware pattern over inheritance/monolithic handlers
**Why:** Enables composable, testable, and reusable request processing logic
**Implementation:** Handler functions wrapped in sequence: `error → timeout → logging → caching → proxy`

### Caching Strategy
**Decision:** LRU cache with TTL over simple map caching
**Why:** Prevents memory leaks, handles concurrent access, respects HTTP cache headers
**Challenge:** Implementing thread-safe eviction while maintaining performance
**Learning:** Go's `sync.RWMutex` provides fine-grained locking for read-heavy workloads

### Configuration Hierarchy
**Decision:** Command-line flags > Environment variables > Config file > Defaults
**Why:** Allows deployment flexibility while maintaining sensible defaults
**Implementation:** Single `LoadConfig()` function with clear precedence rules

### Error Handling Philosophy
**Decision:** Structured JSON errors over plain text responses
**Why:** API consumers need predictable error formats for proper handling
**Implementation:** Custom error types with consistent `{error, message}` structure

### Testing Approach
**Decision:** Unit tests for pure functions, integration tests for HTTP flows
**Why:** Unit tests catch logic bugs fast, integration tests validate real behavior
**Coverage:** ~64% with focus on business logic over standard library wrappers

## 🚀 Running the Project

```bash
git clone https://github.com/josiah-mbao/go-proxy-server.git
cd go-proxy-server
go mod tidy
go run .
```

## Testing

The project includes comprehensive unit and integration tests. Run tests using:

```bash
# Run all tests
make test

# Run tests with verbose output
make test-verbose

# Run tests with coverage report
make test-cover

# Run specific test file
go test -v config_test.go
```

### Test Coverage

Current test coverage: ~64%

### Test Structure

- **config_test.go**: Tests for configuration loading from various sources
- **middleware_test.go**: Tests for request logging middleware
- **proxy_test.go**: Tests for reverse proxy functionality
- **integration_test.go**: End-to-end integration tests
- **testdata/**: Test fixtures and sample configuration files

## Configuration

Edit the server.go file or create a configuration file to customize the proxy server settings such as the port number, backend server address, and caching options.

## Usage

Running the Server
Start the proxy server using the following command:

```bash
go run .
```

By default, the server listens on port 8080. You can configure this using environment variables, command-line flags, or a config file.

### Configuration Options

The proxy server supports multiple configuration methods (in order of precedence):

1. **Command-line flags** (highest priority):
   ```bash
   go run . -port 9090 -backend "http://example.com" -log-level debug
   ```

2. **Environment variables**:
   ```bash
   export PROXY_PORT=9090
   export PROXY_BACKEND="http://example.com"
   export PROXY_LOG_LEVEL=debug
   go run .
   ```

3. **Config file** (JSON):
   ```bash
   go run . -config config.json
   ```
   Or via environment variable:
   ```bash
   export PROXY_CONFIG_FILE=config.json
   go run .
   ```

   Sample `config.json`:
   ```json
   {
     "port": 8080,
     "backend": "http://127.0.0.1:5000",
     "log_level": "info",
     "cache_enabled": true,
     "cache_size": 100,
     "cache_ttl_seconds": 300
   }
   ```

### Caching Configuration

The proxy supports optional response caching to reduce backend load:

- **cache_enabled**: Enable/disable caching (default: false)
- **cache_size**: Maximum number of cached responses (default: 100)
- **cache_ttl_seconds**: Cache entry time-to-live in seconds (default: 300)

**Caching Behavior:**
- Only caches GET requests with 2xx responses
- Respects `Cache-Control: no-cache` and `Cache-Control: private` headers
- Uses LRU (Least Recently Used) eviction policy
- Thread-safe for concurrent access
- Responses include `X-Cache: HIT/MISS/BYPASS` headers

### Rate Limiting Configuration

The proxy includes configurable rate limiting to prevent abuse and ensure fair resource distribution:

- **rate_limit_enabled**: Enable/disable rate limiting (default: false)
- **rate_limit_requests_per_minute**: Maximum requests per minute per client (default: 100)
- **rate_limit_burst_size**: Token bucket capacity for burst requests (default: 20)

**Rate Limiting Features:**
- Token bucket algorithm with configurable refill rates
- Per-client rate limiting based on IP address
- Thread-safe concurrent access
- Automatic cleanup of stale rate limit buckets
- RFC-compliant HTTP headers (X-RateLimit-*, Retry-After)

**Rate Limit Headers:**
- `X-RateLimit-Limit`: Maximum requests per period
- `X-RateLimit-Remaining`: Remaining requests in current period
- `X-RateLimit-Reset`: Time when the limit resets
- `Retry-After`: Seconds to wait before retrying (when limit exceeded)

### Error Handling & Timeouts

The proxy includes comprehensive error handling and timeout management:

- **request_timeout_seconds**: Maximum time for backend requests (default: 30)
- **shutdown_timeout_seconds**: Graceful shutdown timeout (default: 30)

**Error Handling Features:**
- Panic recovery with structured error responses
- Timeout handling for slow backend requests
- Graceful shutdown with signal handling (SIGINT/SIGTERM)
- Structured JSON error responses
- Comprehensive error logging

**Shutdown Process:**
1. Receives interrupt signal (Ctrl+C or SIGTERM)
2. Stops accepting new requests
3. Waits for active requests to complete
4. Times out after shutdown_timeout_seconds
5. Forces shutdown if needed

## 📚 Lessons Learned

### HTTP Server Development
**Goroutines are everywhere:** Every HTTP request spawns a goroutine - understanding this is crucial for resource management and debugging.

**Context is king:** Go's context package is essential for cancellation, timeouts, and request-scoped values. Without it, cleanup becomes nearly impossible.

**Middleware composition:** The pattern of wrapping handlers enables powerful, testable architectures. It's the foundation of most Go web frameworks.

### Production Considerations
**Never trust user input:** All configuration sources need validation. Environment variables can be malformed, JSON files corrupted.

**Graceful shutdown isn't optional:** In production, you can't just kill processes. Proper shutdown prevents data loss and ensures client requests complete.

**Caching is complex:** Beyond basic storage, you need TTL, eviction policies, thread safety, and HTTP semantics compliance.

### Testing Philosophy
**Integration tests catch real bugs:** Unit tests verify logic, but only integration tests validate the complete HTTP flow.

**Test the error paths:** Most production bugs occur during failures, not success cases. Test timeouts, network errors, invalid inputs.

## 💼 What This Project Demonstrates

### Backend Engineering Skills
- **HTTP Protocol Expertise:** Deep understanding of request/response lifecycle, headers, status codes
- **Concurrent Programming:** Goroutines, channels, mutexes for thread-safe operations
- **System Design:** Middleware architecture, caching strategies, configuration management

### Production-Ready Development
- **Error Handling:** Structured error responses, panic recovery, timeout management
- **Observability:** Request logging, metrics, debugging capabilities
- **Reliability:** Graceful shutdown, resource cleanup, signal handling

### Quality Engineering
- **Testing Strategy:** Comprehensive unit and integration test suites
- **Code Architecture:** Modular design, dependency injection, separation of concerns
- **Performance Optimization:** LRU caching, connection pooling, efficient data structures

### DevOps & Deployment
- **Configuration Management:** Multi-environment config with proper precedence
- **Container Readiness:** Signal handling, graceful shutdown for orchestration
- **Monitoring Integration:** Structured logging, error tracking, performance metrics

This project showcases the complete journey from concept to production-ready system, demonstrating both technical depth and engineering maturity.
