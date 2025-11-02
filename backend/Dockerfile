# Build stage
FROM golang:1.25.3-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum* ./

# Copy source code
COPY . .

# Download and tidy modules
RUN go mod download && go mod tidy

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/server .

# Expose port
EXPOSE 8080

# Run the binary
CMD ["./server"]
