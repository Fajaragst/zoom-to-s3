FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o zoom-to-s3 ./cmd/api

# Create a minimal production image
FROM alpine:3.18

WORKDIR /app

# Install CA certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

# Copy the binary from the builder stage
COPY --from=builder /app/zoom-to-s3 .

# Create directory for logs
RUN mkdir -p /app/logs

# Set the entry point
ENTRYPOINT ["/app/zoom-to-s3"] 