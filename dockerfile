# Multi-stage build for Go application optimized for Cloud Run
FROM golang:1.22.4-alpine AS builder

# Install necessary packages
RUN apk add --no-cache git ca-certificates

# Set working directory
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application with optimizations
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-w -s" -o main ./cmd/api/main.go

# Final stage - minimal runtime image
FROM alpine:latest

# Install ca-certificates for HTTPS requests and wget for health check
RUN apk --no-cache add ca-certificates wget

# Create non-root user for security
RUN adduser -D -s /bin/sh appuser

WORKDIR /app

# Copy the binary from builder stage
COPY --from=builder /app/main .

# Copy static files needed by the application
COPY --from=builder /app/websocket-chat-pgx ./websocket-chat-pgx

# Note: Firebase service account is provided via FIREBASE_SERVICE_ACCOUNT_JSON environment variable in production

# Change ownership to non-root user
RUN chown -R appuser:appuser /app

# Switch to non-root user
USER appuser

# Expose port (Cloud Run uses PORT env var, defaults to 8080)
EXPOSE 8080

# Set default port for Cloud Run
ENV PORT=8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=10s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:${PORT}/health || exit 1

# Run the application
CMD ["./main"]