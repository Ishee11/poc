# Kafka Outbox Flow

This project uses Kafka for asynchronous side effects, not as the source of truth.
PostgreSQL remains the transactional source of truth for sessions and operations.

## Flow

1. HTTP use cases write business data and `outbox_events` in the same PostgreSQL transaction.
2. The app background relay reads pending outbox events.
3. The relay publishes events to Kafka topic `poker.events`.
4. Published rows are marked with `published_at`.
5. `cmd/audit-consumer` reads Kafka events and writes them to `audit_events`.

## Local Dev

Start the dev stack:

```bash
docker compose -f docker-compose.dev.yml up -d db kafka app
```

Run the audit consumer from the host:

```bash
DATABASE_URL='postgres://poker:poker@127.0.0.1:15432/poker?sslmode=disable' \
KAFKA_BROKERS='127.0.0.1:19092' \
KAFKA_OUTBOX_TOPIC='poker.events' \
KAFKA_AUDIT_GROUP_ID='poker-audit' \
go run ./cmd/audit-consumer
```

Inspect pending or published outbox rows:

```bash
docker compose -f docker-compose.dev.yml exec db \
  psql -U poker -d poker -c "select id, event_type, published_at, attempts, last_error from outbox_events order by created_at desc limit 20;"
```

Inspect consumed audit rows:

```bash
docker compose -f docker-compose.dev.yml exec db \
  psql -U poker -d poker -c "select event_id, event_type, aggregate_type, aggregate_id, consumed_at from audit_events order by consumed_at desc limit 20;"
```

Consume the Kafka topic directly:

```bash
docker run --rm --network poc_default bitnami/kafka:3.7 \
  kafka-console-consumer.sh --bootstrap-server kafka:9092 --topic poker.events --from-beginning
```

## Event Types

- `operation.created`: emitted for `buy_in` and `cash_out`.
- `operation.reversed`: emitted when an operation is reversed.
- `session.finished`: emitted when a balanced session is finished.

Kafka delivery is at-least-once. Consumers must be idempotent; the audit consumer uses `event_id` as a unique key.
