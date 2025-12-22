# Build stage
FROM node:22-alpine AS frontend-builder

WORKDIR /build

# Copy go mod files
COPY app ./app
RUN npm --prefix ./app install
RUN npm --prefix ./app run build
RUN ls ./internal/server/public

# Build stage
FROM golang:1.25.5-alpine AS backend-builder

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Copy built frontend assets
COPY --from=frontend-builder /build/internal/server/public ./internal/server/public

# Build the application with the same flags as goreleaser
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X autobutler/pkg/util/versionutil.Semver=${VERSION}" \
    -o autobutler ./cmd/autobutler/main.go

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy the binary
COPY --from=backend-builder /build/autobutler .

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
