# Multi-stage build for smaller image
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
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -a -installsuffix cgo \
    -ldflags="-w -s" \
    -o mcphost .

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add \
    ca-certificates \
    curl \
    bash \
    jq \
    && rm -rf /var/cache/apk/*

# Create non-root user
RUN addgroup -g 1001 -S mcphost && \
    adduser -u 1001 -S mcphost -G mcphost

# Set working directory
WORKDIR /home/mcphost

# Copy binary from builder stage
COPY --from=builder /app/mcphost .

# Copy documentation
COPY --from=builder /app/docs/ ./docs/
COPY --from=builder /app/examples/ ./examples/
COPY --from=builder /app/scripts/ ./scripts/

# Copy configuration examples
COPY --from=builder /app/local.json ./config/local.json.example

# Create necessary directories
RUN mkdir -p /home/mcphost/config /home/mcphost/logs /home/mcphost/data

# Set ownership
RUN chown -R mcphost:mcphost /home/mcphost

# Switch to non-root user
USER mcphost

# Set environment variables
ENV MCPHOST_CONFIG_DIR=/home/mcphost/config
ENV MCPHOST_DATA_DIR=/home/mcphost/data
ENV MCPHOST_LOG_DIR=/home/mcphost/logs

# Expose port (if needed for web interface)
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD ./mcphost --version || exit 1

# Set entrypoint
ENTRYPOINT ["./mcphost"]

# Default command
CMD ["--help"]
