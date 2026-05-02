# ========= BUILD =========
FROM golang:1.26.2-alpine AS builder

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

# ========= RUNTIME =========
FROM scratch

WORKDIR /app

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /app/app .

EXPOSE 8080

ENTRYPOINT ["/app/app"]
