# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git ca-certificates

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binaries
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/indexer ./cmd/indexer
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/analytics ./cmd/analytics

# Final stage
FROM alpine:3.19

WORKDIR /app

# Install ca-certificates for HTTPS
RUN apk add --no-cache ca-certificates

# Copy binaries from builder
COPY --from=builder /bin/api /app/api
COPY --from=builder /bin/indexer /app/indexer
COPY --from=builder /bin/analytics /app/analytics

# Copy migrations
COPY migrations /app/migrations

# Default command
CMD ["/app/api"]
