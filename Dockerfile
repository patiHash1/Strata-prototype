# ==========================================
# Build Stage
# ==========================================
FROM golang:1.26.5-alpine AS builder

# Install SSL certificates & build tools
RUN apk add --no-cache ca-certificates git tzdata

WORKDIR /app

# Cache Go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary directly from the entrypoint directory
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o /app/server ./cmd/api

# ==========================================
# Run Stage
# ==========================================
FROM alpine:latest

# Install minimal runtime certificates and timezone data
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Create a non-root user for security
RUN adduser -D -g '' appuser
USER appuser

# Copy built binary from builder
COPY --from=builder /app/server /app/server

EXPOSE 8080

ENTRYPOINT ["/app/server"]