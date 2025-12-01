# ---- Build stage ----
FROM golang:1.25.3 AS builder

WORKDIR /app

# Copy dependencies first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o todo

# ---- Runtime stage ----
FROM alpine:3.22

WORKDIR /app

# Install SQLite and create data directory
RUN apk add --no-cache sqlite && \
	mkdir -p /app/data

# Copy binary and web files
COPY --from=builder /app/todo .
COPY --from=builder /app/web ./web/

EXPOSE ${TODO_PORT:-7540}

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
	CMD wget --no-verbose --tries=1 --spider "http://localhost:${TODO_PORT:-7540}" || exit 1

# Run application
CMD ["./todo"]
