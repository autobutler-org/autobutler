FROM --platform=$BUILDPLATFORM golang:1.21-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG TARGETOS
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o main .

FROM --platform=$TARGETPLATFORM alpine:latest

COPY --from=builder /app/main /usr/local/bin/main
RUN chmod +x /usr/local/bin/main

WORKDIR /app
EXPOSE 8080

CMD ["main"] 