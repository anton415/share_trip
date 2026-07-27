# ShareTrip

ShareTrip is a backend service for managing shared trips. It implements the trip lifecycle from creating a draft to publishing it, while keeping domain rules, persistence, and the HTTP API separated.

## Features

- create a trip in the `draft` status;
- publish a trip with authorization and state checks;
- retrieve a trip by its identifier;
- store trip history and outbox events transactionally;
- verify application and PostgreSQL readiness through `GET /ready`;
- propagate a request identifier through structured logs.

## Technology stack

- Go 1.25;
- Fiber v2;
- PGX v5 and PostgreSQL 16;
- Goose migrations;
- Testcontainers and Testify;
- Docker Compose;
- Loki, Grafana, and Alloy for local observability.

## Requirements

- Go 1.25 or newer;
- GNU Make;
- Docker with Docker Compose;
- `curl` for the end-to-end readiness check.

## Configuration

Default local values match the PostgreSQL service from `deploy/docker-compose.yml`. The available environment variables are documented in `configs/local.env.example`:

```text
DB_HOST
DB_PORT
DB_USER
DB_PASSWORD
DB_NAME
DB_SSLMODE
DATABASE_URL
HTTP_ADDR
```

Export these variables before starting the application when you need values other than the defaults.

## Local start

Install project tools and download dependencies:

```bash
make deps
```

Start PostgreSQL and apply migrations:

```bash
make up
make migrate-up
```

Run the service:

```bash
make run
```

The service listens on `http://localhost:8080` by default. In another terminal, verify readiness:

```bash
make e2e
```

Stop the local infrastructure:

```bash
make down
```

## HTTP API

| Method | Path | Description |
| --- | --- | --- |
| `GET` | `/ready` | Checks the service and PostgreSQL connection. |
| `POST` | `/trip/create` | Creates a trip in the `draft` status. |
| `POST` | `/trip/publish` | Publishes a draft trip. |
| `GET` | `/trip/:id` | Returns a trip by its identifier. |

### Трассировка требований публикации

| Требование | Реализация | Интеграционный тест |
| --- | --- | --- |
| Вернуть `403`, если `clientId != trip.DriverID` | [Проверка доменного правила](internal/domain/publish_trip.go#L30-L37), [преобразование в HTTP-ответ](internal/api/trip.go#L129-L133) | [Сценарий Forbidden](internal/api/publish_trip_test.go#L95-L106) |
| Вернуть `404`, если поездки нет | [Преобразование `pgx.ErrNoRows`](internal/repositories/trip.go#L143-L145), [преобразование в HTTP-ответ](internal/api/trip.go#L124-L128) | [Сценарий Not Found](internal/api/publish_trip_test.go#L108-L117) |
| Вернуть `409`, если поездка не в статусе `draft` | [Проверка доменного правила](internal/domain/publish_trip.go#L47-L54), [преобразование в HTTP-ответ](internal/api/trip.go#L134-L138) | [Сценарий Conflict](internal/api/publish_trip_test.go#L119-L137) |
| Вернуть `204`, если поездка уже в статусе `published` | [Проверка доменного правила](internal/domain/publish_trip.go#L39-L45), [преобразование в HTTP-ответ](internal/api/trip.go#L171-L173) | [Сценарий No Content](internal/api/publish_trip_test.go#L139-L166) |

## Checks

```bash
make fmt
make lint
make test
make coverage
make build
make check
```

Integration tests start an isolated PostgreSQL 16 container, so Docker must be running.
