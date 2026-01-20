.PHONY: build test test-verbose test-cover clean run docker-build docker-run

# Build the application
build:
	go build -o reverse-proxy .

# Run tests
test:
	go test ./...

# Run tests with verbose output
test-verbose:
	go test -v ./...

# Run tests with coverage
test-cover:
	go test -cover ./...

# Clean build artifacts
clean:
	rm -f reverse-proxy
	go clean

# Run the application
run:
	go run .

# Run with specific config
run-config:
	go run . -config config.json

# Docker commands
docker-build:
	docker build -t reverse-proxy .

docker-run:
	docker run -p 8080:8080 reverse-proxy

# Development helpers
fmt:
	go fmt ./...

vet:
	go vet ./...

lint: fmt vet test

# CI/CD
ci: lint test-cover
