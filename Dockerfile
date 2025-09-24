# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache gcc musl-dev

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o monitor ./cmd/monitor

# Runtime stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests and sqlite dependencies
RUN apk --no-cache add ca-certificates sqlite

# Create app directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/monitor .

# Create directory for database
RUN mkdir -p /app/data

# Expose web port
EXPOSE 8080

# Set default database path (can be overridden by volume mount)
ENV DB_PATH=/app/data/network_monitor.db

# Run the application
CMD ["./monitor", "-db", "/app/data/network_monitor.db"]
