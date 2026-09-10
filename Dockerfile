# Multi-stage build for minimal image size
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy dependency manifests
COPY go.mod go.sum ./
RUN go mod download

# Copy source files
COPY server/ ./server/

# Compile statically linked binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o snake-server ./server

# Final production image
FROM alpine:latest

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/snake-server .

# Copy static frontend client assets
COPY client/ ./client/

# Expose standard port
EXPOSE 8080

# Run server
CMD ["./snake-server"]
