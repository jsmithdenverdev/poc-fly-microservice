# Build stage
FROM golang:1.23.4-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache gcc musl-dev git

# Install Delve debugger
RUN go install github.com/go-delve/delve/cmd/dlv@latest

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application with debug information
RUN CGO_ENABLED=0 GOOS=linux go build -gcflags="all=-N -l" -o main .

# Final stage
FROM alpine:latest

WORKDIR /app

# Install necessary runtime dependencies
RUN apk add --no-cache libc6-compat

# Copy the binary and debugger from builder
COPY --from=builder /app/main .
COPY --from=builder /go/bin/dlv .

EXPOSE 8080 40000

# Start the debugger
CMD ["./dlv", "--listen=:40000", "--headless=true", "--api-version=2", "--accept-multiclient", "exec", "./main"]
