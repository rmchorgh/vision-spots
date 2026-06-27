FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /vision-spots-backend .

FROM gcr.io/distroless/static-debian12

COPY --from=builder /vision-spots-backend /vision-spots-backend

USER 65532:65532

EXPOSE 5055

HEALTHCHECK --interval=30s --timeout=3s \
  CMD ["/vision-spots-backend", "-health"]

ENTRYPOINT ["/vision-spots-backend"]
