# Build stage
FROM --platform=$BUILDPLATFORM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application for ARM64
RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o main .

# Final stage
FROM --platform=$TARGETPLATFORM alpine:latest

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/main .

# Copy any additional files needed at runtime
COPY .env* ./

# Make the binary executable
RUN chmod +x main

# Expose the port the app runs on
EXPOSE 8080

# Command to run the executable
CMD ["./main"] 