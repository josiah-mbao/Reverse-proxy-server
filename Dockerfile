# Use the official Go image as a builder
FROM golang:1.23.4 AS builder

# Set the working directory
WORKDIR /app

# Copy the Go module files and download dependencies
COPY go.mod ./
RUN go mod download

# Copy the source code
COPY . .

# Build the Go binary (ensure it's built for Linux)
RUN GOOS=linux GOARCH=arm64 go build -o reverse-proxy main.go

# Use Ubuntu as the final image to avoid GLIBC issues
FROM ubuntu:latest

# Install required dependencies (GLIBC)
RUN apt update && apt install -y libc6

# Set the working directory
WORKDIR /

# Copy the compiled binary from the builder stage
COPY --from=builder /app/reverse-proxy .

# Expose port 8080
EXPOSE 8080

# Command to run the application
CMD ["./reverse-proxy"]
