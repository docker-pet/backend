# --- Build stage ---
FROM golang:1.24-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

# Copy go.mod and go.sum for dependency caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source files
COPY . .

# Build the binary (replace main.go with your entrypoint if needed)
RUN CGO_ENABLED=0 go build -ldflags "-X 'main.Version=$(git describe --tags)' -X 'main.Commit=$(git rev-parse HEAD)' -X 'main.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)'" -o backend .

# --- Final stage ---
FROM alpine:3.20

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/backend /app/backend

EXPOSE 80

CMD ["/app/backend", "serve", "--http=0.0.0.0:80"]