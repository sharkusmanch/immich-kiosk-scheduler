# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build arguments for version info
ARG VERSION=dev
ARG COMMIT=none
ARG BUILD_DATE=unknown

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.buildDate=${BUILD_DATE}" \
    -o /immich-kiosk-scheduler \
    ./cmd/immich-kiosk-scheduler

# Runtime stage
FROM scratch

# Copy CA certificates for HTTPS
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy binary
COPY --from=builder /immich-kiosk-scheduler /immich-kiosk-scheduler

# Default port
EXPOSE 8080

# Run as non-root
USER 65534:65534

ENTRYPOINT ["/immich-kiosk-scheduler"]
CMD ["serve"]
