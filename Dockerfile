# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-w -s" -o connex ./cmd/server

# Production stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1001 -S connex && \
    adduser -u 1001 -S connex -G connex

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/connex .

# Copy any additional files (migrations, configs, etc.)
COPY --from=builder /app/internal/db/migrations ./internal/db/migrations
COPY --from=builder /app/scripts ./scripts

# Change ownership to non-root user
RUN chown -R connex:connex /app

# Switch to non-root user
USER connex

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the application
CMD ["./connex"] 