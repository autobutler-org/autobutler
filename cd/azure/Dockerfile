# Build stage
FROM golang:1.24.4-alpine AS builder

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application with the same flags as goreleaser
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X autobutler/internal/version.Semver=${VERSION}" \
    -o autobutler .

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /build/autobutler .

# Copy static assets
COPY --from=builder /build/internal/server/public ./internal/server/public

# Create data directory for volume mount
RUN mkdir -p ${HOME}/autobutler/data

# Set environment variables
ENV PORT=8080
ENV GIN_MODE=release

# Expose the port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/ || exit 1

# Run the application
CMD ["./autobutler", "serve"]
