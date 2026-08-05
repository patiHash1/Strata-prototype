# ==========================================
# Build Stage
# ==========================================
FROM golang:1.23-alpine AS builder

# Install SSL certificates & build tools
RUN apk add --no-cache ca-certificates git tzdata

WORKDIR /app

# Cache Go modules (only re-runs when dependencies change)
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build statically-linked binary (CGO_ENABLED=0 for lightweight scratch/alpine execution)
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o /app/server .

# ==========================================
# Run Stage
# ==========================================
FROM alpine:latest

# Install minimal runtime certificates and timezone data
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Create a non-root user for security best practices
RUN adduser -D -g '' appuser
USER appuser

# Copy built binary from the builder stage
COPY --from=builder /app/server /app/server

# Informational port (Railway will override this dynamically using $PORT)
EXPOSE 8080

# Execute binary
ENTRYPOINT ["/app/server"]