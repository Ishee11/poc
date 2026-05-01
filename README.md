# Poker Session Control

Веб-приложение и HTTP API для учёта покерных cash-game сессий.

Проект хранит игроков, сессии, buy-in/cash-out операции, пересчитывает фишки и денежные итоги по курсу сессии, показывает статистику по игрокам и сессиям.

## Что Уже Есть

- Встроенный web UI на `/`.
- HTTP API на стандартном `net/http`.
- PostgreSQL как основное хранилище.
- Автоматический запуск миграций при старте приложения.
- Идемпотентность write-операций через `request_id` в JSON body.
- `X-Request-ID` для трассировки HTTP-запросов.
- Structured JSON logs через `log/slog`.
- Prometheus metrics на `/metrics`.
- Swagger UI на `/swagger/`.

## Быстрый Старт

Запустить PostgreSQL:

```sh
docker compose up -d db
```

Запустить приложение локально:

```sh
DATABASE_URL='postgres://poker:poker@127.0.0.1:5432/poker?sslmode=disable' \
HTTP_PORT=8080 \
LOG_LEVEL=info \
HTTP_ACCESS_LOG=errors \
go run ./cmd/app
```

Открыть UI:

```text
http://127.0.0.1:8080/
```

Проверить healthcheck:

```sh
curl -sS http://127.0.0.1:8080/health
```

Запустить приложение и БД полностью в Docker:

```sh
docker compose up --build -d
```

Локально поднять Prometheus и Grafana:

```sh
docker compose -f docker-compose.observability.yml up -d
```

После старта Grafana уже содержит готовый dashboard `Poker App Overview` и datasource `Tempo` для HTTP/usecase/PostgreSQL traces.

Grafana берет admin credentials и внешний URL из `.env.observability`.
Alertmanager и Telegram alerting тоже настраиваются через `.env.observability`.
По умолчанию этот стек смотрит на dev-приложение `host.docker.internal:18080`.
Для prod нужно переключить `PROMETHEUS_CONFIG_FILE=prometheus.prod.yml` и выдать отдельные внешние порты для Grafana и alerting.

## Конфигурация

Минимальные переменные окружения:

| Переменная | Обязательная | По умолчанию | Описание |
| --- | --- | --- | --- |
| `DATABASE_URL` | да | - | PostgreSQL DSN. |
| `HTTP_PORT` | нет | `8080` | Порт HTTP сервера. |
| `LOG_LEVEL` | нет | `info` | Уровень `slog`: `debug`, `info`, `warn`, `error`. |
| `HTTP_ACCESS_LOG` | нет | `errors` | Режим HTTP access log: `errors`, `all`, `off`. |
| `OTEL_SERVICE_NAME` | нет | `poker-app` | Имя сервиса в OpenTelemetry traces. |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | нет | - | OTLP HTTP endpoint для traces, например `tempo:4318` в Docker или `http://127.0.0.1:4318/v1/traces` локально. Если пусто, tracing exporter выключен. |
| `OTEL_EXPORTER_OTLP_INSECURE` | нет | `true` | Использовать plaintext OTLP HTTP. |

Продовый дефолт для `HTTP_ACCESS_LOG` - `errors`: успешные `2xx/3xx` запросы не шумят в логах, пишутся только `4xx/5xx`.

Для локальной отладки можно включить полный поток:

```sh
HTTP_ACCESS_LOG=all
```

## Логи И Ошибки

Логи пишутся в JSON через `slog`.

Обычный production access log появляется только для ошибок:

```json
{
  "level": "WARN",
  "msg": "http_request",
  "request_id": "req-...",
  "method": "GET",
  "path": "/stats/player",
  "status": 400,
  "duration_ms": 1.2,
  "trace_id": "abc123...",
  "span_id": "def456...",
  "bytes": 74,
  "error_code": "player_id_required"
}
```

При panic приложение возвращает JSON `internal_error`, логирует `panic_recovered` со stack trace и тот же `request_id`.

Формат ошибок API:

```json
{
  "error": "player_not_found",
  "request_id": "req-..."
}
```

Если ошибка содержит детали, они возвращаются в `details`, например для несбалансированной сессии:

```json
{
  "error": "session_not_balanced",
  "request_id": "req-...",
  "details": {
    "remaining_chips": 100
  }
}
```

## Основные Сценарии

1. Создать игрока: `POST /players`.
2. Создать сессию: `POST /sessions/start`.
3. Добавить buy-in: `POST /operations/buy-in`.
4. Сделать cash-out: `POST /operations/cash-out`.
5. При ошибочной операции выполнить reverse: `POST /operations/reverse`.
6. Завершить сессию: `POST /sessions/finish`.

Сессию можно завершить только когда сумма buy-in равна сумме cash-out.

## Основные Endpoint'ы

| Метод | Путь | Назначение |
| --- | --- | --- |
| `GET` | `/health` | Healthcheck. |
| `GET` | `/swagger/` | Swagger UI. |
| `GET` | `/players` | Список игроков. |
| `POST` | `/players` | Создание игрока. |
| `POST` | `/sessions/start` | Создание сессии. |
| `POST` | `/sessions/finish` | Завершение сессии. |
| `GET` | `/sessions?session_id=...` | Карточка сессии. |
| `GET` | `/sessions/players?session_id=...` | Игроки в сессии. |
| `GET` | `/sessions/operations?session_id=...` | Операции сессии. |
| `POST` | `/operations/buy-in` | Buy-in. |
| `POST` | `/operations/cash-out` | Cash-out. |
| `POST` | `/operations/reverse` | Отмена операции. |
| `GET` | `/stats/sessions` | Сводка по сессиям. |
| `GET` | `/stats/players` | Сводка по игрокам. |
| `GET` | `/stats/player?player_id=...` | Детальная статистика игрока. |

Актуальная схема API генерируется в `docs/swagger.yaml` и доступна через `/swagger/`.

## Тесты

Запустить все тесты:

```sh
go test ./...
```

Доступные helper-команды:

```sh
make help
```

Для integration-тестов с PostgreSQL нужен локальный DSN:

```sh
DATABASE_URL='postgres://poker:poker@127.0.0.1:5432/poker?sslmode=disable' \
go test ./internal/controller/http ./internal/infra/postgres
```

Тесты с БД отказываются работать с нелокальным host, если явно не выставить:

```sh
ALLOW_DESTRUCTIVE_INTEGRATION_TESTS=true
```

## Структура

```text
cmd/app                 entrypoint приложения
internal/app            сборка приложения, конфиг, DB, HTTP server
internal/controller/http HTTP handlers, middleware, DTO, error mapping
internal/usecase        бизнес-сценарии
internal/entity         доменная модель и доменные ошибки
internal/infra/postgres PostgreSQL repositories and migrations
web                     embedded frontend
docs                    swagger, схемы и эксплуатационная документация
```

## Дополнительная Документация

- [Runbook](docs/RUNBOOK.md) - эксплуатация, логи, проверки и типовые проблемы.
- [POC notes](README_POC.md) - исторические заметки по доменной модели и решениям.
