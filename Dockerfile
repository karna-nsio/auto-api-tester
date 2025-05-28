# Build stage
FROM golang:1.24.3-alpine AS builder

# Set working directory
WORKDIR /app

# Install git and build dependencies
RUN apk add --no-cache git

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o auto-api-tester .

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

# Set working directory
WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/auto-api-tester .

# Create directories for testdata and reports
RUN mkdir -p /app/testdata /app/reports

# Set environment variables
ENV AUTH_TOKEN=""

# Set entrypoint
ENTRYPOINT ["./auto-api-tester"]

# Default command
CMD ["--help"] 