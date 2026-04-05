# Build stage
FROM golang:1.25.6-alpine AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the Go application static binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o engine ./cmd/engine

# Final minimal execution stage
FROM alpine:latest

WORKDIR /app

# Import from builder
COPY --from=builder /app/engine .

# Install CA certificates for TLS (required by Binance WSS)
RUN apk --no-cache add ca-certificates

# Expose standard Prometheus metrics port (Phase 5 prep)
EXPOSE 8080

ENTRYPOINT ["./engine"]
