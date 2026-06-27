# Multi-stage Dockerfile for vision-spots backend
# Final image is distroless for security + small size

FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go mod and sum first for better caching
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build static binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /vision-spots-backend .

# Final stage - distroless for minimal attack surface
FROM gcr.io/distroless/static-debian12

COPY --from=builder /vision-spots-backend /vision-spots-backend

# Run as non-root
USER 65532:65532

EXPOSE 5055

# Healthcheck used by orchestration / selfhost-dns
HEALTHCHECK --interval=30s --timeout=3s \
  CMD ["/vision-spots-backend", "-health"] || exit 1

ENTRYPOINT ["/vision-spots-backend"]
