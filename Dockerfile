# ========= BUILD =========
FROM golang:1.25.0-alpine AS builder

WORKDIR /app

# TLS certificates for outbound HTTPS in the scratch runtime image.
RUN apk add --no-cache ca-certificates

# зависимости
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

# код
COPY . .

# билд (static)
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -o app ./cmd/app/main.go

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -o healthcheck ./cmd/healthcheck/main.go

# ========= RUNTIME =========
FROM scratch

WORKDIR /app

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /app/app .
COPY --from=builder /app/healthcheck .

EXPOSE 8080

HEALTHCHECK --interval=10s --timeout=3s --retries=3 CMD ["/app/healthcheck"]

ENTRYPOINT ["/app/app"]
