# syntax=docker/dockerfile:1

# Base build stage
FROM golang:1.23.4-alpine AS base-builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache gcc musl-dev git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Debug build stage
FROM base-builder AS debug-builder
RUN go install github.com/go-delve/delve/cmd/dlv@latest
RUN CGO_ENABLED=0 GOOS=linux go build -gcflags="all=-N -l" -o main .

# Development build stage
FROM base-builder AS dev-builder
RUN CGO_ENABLED=0 GOOS=linux go build -o main .

# Production build stage
FROM base-builder AS prod-builder
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o main .

# Debug image
FROM alpine:latest AS debug
WORKDIR /app
RUN apk add --no-cache libc6-compat
COPY --from=debug-builder /app/main .
COPY --from=debug-builder /go/bin/dlv .
ENV LOG_LEVEL=debug
EXPOSE 8080 40000
CMD ["./dlv", "--listen=:40000", "--headless=true", "--api-version=2", "--accept-multiclient", "exec", "./main"]

# Development image
FROM alpine:latest AS dev
WORKDIR /app
RUN apk add --no-cache ca-certificates tzdata
COPY --from=dev-builder /app/main .
ENV LOG_LEVEL=debug
EXPOSE 8080
CMD ["./main"]

# Production image
FROM scratch AS prod
COPY --from=prod-builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=prod-builder /usr/share/zoneinfo /usr/share/zoneinfo
WORKDIR /app
COPY --from=prod-builder /app/main .
ENV LOG_LEVEL=error
EXPOSE 8080
CMD ["./main"]
