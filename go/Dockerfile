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

# Install CA certificates for TLS (required by Binance WSS)
RUN apk --no-cache add ca-certificates

# Create a non-root user to run the application
RUN adduser -D appuser
WORKDIR /app

# Import the binary from the builder stage
COPY --from=builder /app/engine .
# Ensure the binary is owned by the appuser
RUN chown appuser:appuser /app/engine

# Switch to the non-root user
USER appuser

# Expose standard Prometheus metrics port
EXPOSE 8080

ENTRYPOINT ["./engine"]
