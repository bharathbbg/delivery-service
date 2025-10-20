FROM golang:1.19-alpine as builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o delivery-service ./cmd/server/main.go

# Final stage
FROM alpine:3.16

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/delivery-service .

# Install CA certificates
RUN apk --no-cache add ca-certificates

# Set user to non-root
RUN adduser -D -g '' appuser
USER appuser

# Expose HTTP and gRPC ports
EXPOSE 8081 50052

# Run the application
CMD ["./delivery-service"]