# ========================================
# BASE STAGE
# ========================================
FROM golang:1.23.4-alpine AS base
WORKDIR /app

RUN apk add --no-cache git

# ========================================
# DEVELOPMENT STAGE
# ========================================
FROM base AS dev

RUN apk add --no-cache bash
RUN go install github.com/go-delve/delve/cmd/dlv@latest

COPY . .
RUN go mod download

EXPOSE 40000 8080
CMD ["dlv", "debug", "--headless", "--listen=:40000", "--api-version=2", "--log", "./main.go"]

# ========================================
# PRODUCTION STAGE
# ========================================
FROM base AS prod

# Copy go.mod first for better layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy all source files
COPY *.go ./

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o main .

# ========================================
# FINAL STAGE
# ========================================
FROM scratch AS final
WORKDIR /app

COPY --from=prod /app/main .
EXPOSE 8080
ENTRYPOINT ["/app/main"]
