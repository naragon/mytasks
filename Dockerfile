# Build stage
FROM golang:1.22-alpine AS builder

# Install build dependencies for CGO (required by sqlite3)
RUN apk add --no-cache gcc musl-dev

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-w -s" -o mytasks .

# Runtime stage
FROM alpine:3.19

# Install runtime dependencies for sqlite3
RUN apk add --no-cache ca-certificates sqlite-libs

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/mytasks .

# Create data directory
RUN mkdir -p /data

# Environment variables
ENV PORT=8080
ENV DB_PATH=/data/mytasks.db

# Expose port
EXPOSE 8080

# Run the application
CMD ["./mytasks"]
