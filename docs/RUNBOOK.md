# Runbook

Краткая инструкция по эксплуатации Poker Session Control.

## Сервисы

Приложение состоит из двух обязательных компонентов:

- `poker-app` - HTTP API и embedded web UI.
- `poker-db` - PostgreSQL 16.

Docker Compose поднимает оба сервиса:

```sh
docker compose up --build -d
```

Проверить состояние:

```sh
docker compose ps
curl -sS http://127.0.0.1:8080/health
```

Остановить:

```sh
docker compose down
```

## Метрики И Локальная Grafana

Приложение отдает Prometheus-метрики на:

```text
/metrics
```

Локальный observability-стек:

```sh
docker compose -f docker-compose.observability.yml up -d
```

Grafana читает admin credentials и внешний URL из `.env.observability`.
Alertmanager читает Telegram-настройки из `.env.observability`.

По умолчанию Prometheus скрапит dev-приложение на:

```text
http://host.docker.internal:18080/metrics
```

Порты локального стека:

- Prometheus: `http://127.0.0.1:9090`
- Alertmanager: `http://127.0.0.1:9093`
- Grafana: `http://193.238.134.58:3000`

Логин Grafana по умолчанию:

```text
<значение GF_SECURITY_ADMIN_USER> / <значение GF_SECURITY_ADMIN_PASSWORD>
```

После первого старта в Grafana уже будет готовый dashboard:

```text
Poker / Poker App Overview
```

Если локальное приложение слушает не `18080`, нужно изменить target в `deploy/observability/prometheus/prometheus.yml`.

Если Grafana должна открываться по другому IP или домену, обновить значения в `.env.observability`.

Если пароль содержит символ `$`, в `.env.observability` его нужно писать как `$$`.

Для Telegram-алертов заполнить в `.env.observability`:

```text
TELEGRAM_BOT_TOKEN=...
TELEGRAM_CHAT_ID=...
TELEGRAM_SEND_RESOLVED=true
```

Базовые алерты:

- `AppDown`
- `PrometheusTargetDown`
- `High5xxRate`
- `HighLatencyP95`

## Конфигурация

Обязательная переменная:

```text
DATABASE_URL=postgres://poker:poker@db:5432/poker?sslmode=disable
```

Рекомендуемый production-like минимум:

```text
HTTP_PORT=8080
LOG_LEVEL=info
HTTP_ACCESS_LOG=errors
```

Режимы `HTTP_ACCESS_LOG`:

| Значение | Поведение |
| --- | --- |
| `errors` | Дефолт. Пишет только `4xx/5xx` HTTP-запросы. |
| `all` | Пишет все HTTP-запросы. Использовать для локальной диагностики. |
| `off` | Полностью выключает HTTP access log. |

`LOG_LEVEL=debug` включает debug-уровень и source в `slog`.

## Логи

Смотреть логи Docker:

```sh
docker compose logs -f app
```

Основные события:

| `msg` | Значение |
| --- | --- |
| `server_started` | HTTP server запущен. |
| `server_stopping` | Получен SIGINT/SIGTERM. |
| `server_stopped` | Graceful shutdown завершен. |
| `db_connected` | Подключение к PostgreSQL успешно. |
| `db_connect_retry` | Повтор подключения к PostgreSQL. |
| `db_migrations_applied` | Миграции применены. |
| `http_request` | HTTP request log. По умолчанию только ошибки. |
| `request_error` | Непредвиденная ошибка обработчика, обычно `500`. |
| `panic_recovered` | Panic пойман recovery middleware. |

Ключевые поля для поиска инцидента:

- `request_id` - связывает ответ API, access log и error log.
- `error_code` - стабильный код ошибки для UI/API.
- `status` - HTTP status.
- `path`, `method`, `query` - где произошла ошибка.
- `err` - исходная server-side ошибка, пишется только в логи.

## Проверка После Деплоя

1. Проверить healthcheck:

```sh
curl -sS http://127.0.0.1:8080/health
```

Ожидаемый ответ:

```text
ok
```

2. Открыть UI:

```text
http://127.0.0.1:8080/
```

3. Проверить Swagger:

```text
http://127.0.0.1:8080/swagger/
```

4. Проверить, что в логах есть:

```text
server_started
db_connected
db_migrations_applied
```

## Типовые Проблемы

### `DATABASE_URL is required`

Приложение запущено без DSN.

Проверить env:

```sh
printenv DATABASE_URL
```

Для локального запуска:

```sh
DATABASE_URL='postgres://poker:poker@127.0.0.1:5432/poker?sslmode=disable' go run ./cmd/app
```

### Приложение пишет `db_connect_retry`

PostgreSQL недоступен или DSN неверный.

Проверить контейнер:

```sh
docker compose ps db
docker compose logs db
```

Проверить локальное подключение:

```sh
docker compose up -d db
```

### UI получил `internal_error`

1. Найти `request_id` в ответе API или UI.
2. Найти в логах запись `request_error` или `panic_recovered` с этим `request_id`.
3. Смотреть поля `err`, `panic`, `stack`.

### Сессия Не Завершается

Для завершения сессии сумма buy-in должна быть равна сумме cash-out.

Если ответ:

```json
{
  "error": "session_not_balanced",
  "details": {
    "remaining_chips": 100
  }
}
```

Нужно сделать cash-out или reverse операций, пока остаток на столе не станет `0`.

## Безопасность Данных

Локальные integration-тесты с БД очищают таблицы. По умолчанию они запускаются только против `localhost`, `127.0.0.1` или `::1`.

Не выставлять для production БД:

```text
ALLOW_DESTRUCTIVE_INTEGRATION_TESTS=true
```

## Команды Для Разработки

Запустить тесты:

```sh
go test ./...
```

Проверить frontend JavaScript без сборщика:

```sh
node --check web/js/app.js
node --check web/js/ui/lobby.js
node --check web/js/ui/session.js
node --check web/js/ui/player.js
```

Обновить зависимости:

```sh
go mod tidy
go mod verify
```
