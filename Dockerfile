# ========= BUILD =========
FROM golang:1.26.2-alpine AS builder

WORKDIR /app

# зависимости
COPY go.mod go.sum ./
RUN go mod download

# 👇 ВАЖНО: ломаем кеш перед копированием кода
ARG CACHE_BUST=1

# код
COPY . .

# билд (static)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -a -o app ./cmd/app/main.go

# ========= RUNTIME =========
FROM gcr.io/distroless/base-debian12

WORKDIR /app

COPY --from=builder /app/app .

EXPOSE 8080

ENTRYPOINT ["/app/app"]
