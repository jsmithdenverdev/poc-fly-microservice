FROM golang:1.23.4-alpine

WORKDIR /app

# Install build dependencies and delve
RUN apk add --no-cache gcc musl-dev git
RUN go install github.com/go-delve/delve/cmd/dlv@latest

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application with debug information
RUN CGO_ENABLED=0 GOOS=linux go build -gcflags="all=-N -l" -o main .

# Use a minimal alpine image for the final container
FROM alpine:latest

WORKDIR /app
COPY --from=0 /app/main .
COPY --from=0 /app/dlv .

EXPOSE 8080 40000

# Use delve to run the application
CMD ["dlv", "--listen=:40000", "--headless=true", "--log=true", "--accept-multiclient", "--api-version=2", "exec", "./main"]
