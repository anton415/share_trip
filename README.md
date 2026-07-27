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

### Publish trip responses

`POST /trip/publish` returns:

| Condition | Status | Response |
| --- | --- | --- |
| The trip is in `draft` and `clientId` matches its driver | `200 OK` | JSON containing `tripId` |
| The trip is already `published` | `204 No Content` | Empty body |
| `clientId` does not match the trip driver | `403 Forbidden` | `FORBIDDEN` error |
| The trip does not exist | `404 Not Found` | `NOT_FOUND` error |
| The trip is neither `draft` nor `published` | `409 Conflict` | `CONFLICT` error |

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
