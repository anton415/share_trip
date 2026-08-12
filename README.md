# ShareTrip

ShareTrip — серверный сервис для управления совместными поездками. Он реализует жизненный цикл поездки от создания черновика до публикации и разделяет доменные правила, слой хранения данных и HTTP-интерфейс.

## Возможности

- создание поездки в статусе `draft`;
- публикация поездки с проверкой прав доступа и текущего статуса;
- получение поездки по идентификатору;
- транзакционное сохранение истории поездки и событий в таблице исходящих сообщений (`outbox`);
- проверка готовности приложения и PostgreSQL через `GET /ready`;
- экспорт runtime- и прикладных метрик через `GET /metrics`;
- передача идентификатора запроса в структурированные логи.

## Технологический стек

- Go 1.25;
- Fiber v2;
- PGX v5 и PostgreSQL 16;
- миграции Goose;
- Testcontainers и Testify;
- Docker Compose;
- Prometheus, PostgreSQL Exporter, Grafana, Loki и Alloy для локального мониторинга.

## Требования

- Go 1.25 или новее;
- GNU Make;
- Docker с Docker Compose;
- `curl` для сквозной проверки готовности приложения.

## Конфигурация

Локальные значения по умолчанию соответствуют сервису PostgreSQL из `deploy/docker-compose.yml`. Доступные переменные окружения описаны в `configs/local.env.example`:

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

Если нужны значения, отличные от настроек по умолчанию, экспортируйте эти переменные перед запуском приложения.

## Локальный запуск

Установите инструменты проекта и загрузите зависимости:

```bash
make deps
```

Запустите PostgreSQL и примените миграции:

```bash
make up
make migrate-up
```

Запустите сервис:

```bash
make run
```

По умолчанию сервис доступен по адресу `http://localhost:8080`. В другом терминале проверьте его готовность:

```bash
make e2e
```

## Метрики

После `make up` локальная инфраструктура мониторинга доступна по адресам:

| Компонент | Адрес | Назначение |
| --- | --- | --- |
| Prometheus | `http://localhost:9090` | Сбор и выполнение запросов к метрикам. |
| PostgreSQL Exporter | `http://localhost:9187/metrics` | Экспорт метрик PostgreSQL. |
| Grafana | `http://localhost:3000` | Визуализация метрик. Для первого запуска с чистым volume: `admin` / `admin`. |

Приложение запускается на хосте командой `make run`. Prometheus обращается к нему через `host.docker.internal:8080`, поэтому target `sharetrip` станет `UP` только после запуска приложения. Источник данных Prometheus в Grafana создаётся автоматически из файла конфигурации.

Состояние сбора можно проверить на странице `http://localhost:9090/targets`. Все три target-а — `prometheus`, `sharetrip` и `postgres` — должны иметь состояние `UP`. Минимальные проверочные запросы в Prometheus:

```promql
up{job="sharetrip"}
pg_up{job="postgres"}
go_goroutines{job="sharetrip"}
```

Grafana автоматически загружает в папку `ShareTrip` три dashboard-а:

- `ShareTrip Runtime Go` — goroutines, память, GC, CPU и HTTP RPS;
- `ShareTrip PostgreSQL` — доступность, соединения, размер базы, commits и rollbacks;
- `ShareTrip Application` — HTTP RPS и p95, результаты операций с поездками и p95 репозитория.

Dashboard-ы хранятся как код и недоступны для сохранения изменений через UI. Панели бизнес-процессов и репозитория появляются после соответствующих HTTP-запросов и двух циклов сбора Prometheus. При отсутствии операций в выбранном временном диапазоне p95 не равен нулю — данных для вычисления квантиля ещё нет.

Остановите локальную инфраструктуру:

```bash
make down
```

## HTTP-интерфейс

| Метод | Путь | Описание |
| --- | --- | --- |
| `GET` | `/ready` | Проверяет готовность сервиса и соединение с PostgreSQL. |
| `GET` | `/metrics` | Возвращает runtime- и прикладные метрики в формате Prometheus. |
| `POST` | `/trip/create` | Создаёт поездку в статусе `draft`. |
| `POST` | `/trip/publish` | Публикует черновик поездки. |
| `GET` | `/trip/:id` | Возвращает поездку по идентификатору. |

## Структура пакета API

В пакете `internal/api` каждый HTTP-сценарий находится в отдельном файле. Его DTO, преобразование доменной модели в HTTP-ответ и обработчик расположены вместе. Создание сервера, регистрация маршрутов и общий формат ошибок вынесены отдельно:

| Файл | Назначение |
| --- | --- |
| [`create_trip.go`](internal/api/create_trip.go) | DTO, преобразование ответа и обработчик создания поездки. |
| [`move_trip_draft_to_published.go`](internal/api/move_trip_draft_to_published.go) | DTO и обработчик перевода поездки из `draft` в `published`. |
| [`get_trip_by_id.go`](internal/api/get_trip_by_id.go) | DTO, преобразование ответа и обработчик получения поездки по идентификатору. |
| [`ready.go`](internal/api/ready.go#L11) | Обработчик проверки готовности сервиса. |
| [`metrics.go`](internal/api/metrics.go) | Адаптер обработчика метрик Prometheus для Fiber. |
| [`error_response.go`](internal/api/error_response.go) | Общий DTO ошибки и преобразование ошибок сервиса в HTTP-ответы. |
| [`server.go`](internal/api/server.go#L12) | Зависимости и создание сервера, а также регистрация маршрутов. |
| [`helpers_test.go`](internal/api/helpers_test.go#L18) | Общая подготовка тестовых данных и функции проверок. |

Go компилирует все файлы `.go`, не относящиеся к тестам и объявляющие `package api`, как единый пакет. Поэтому обработчики могут вызывать закрытую функцию `writeFiberServiceError` из `error_response.go`. В общий файл вынесен только действительно общий HTTP-контракт ошибки. Успешные DTO и их преобразователи остаются в файлах соответствующих ручек, чтобы контракты создания и получения поездки могли изменяться независимо. По аналогии `helpers_test.go` содержит функции, общие для нескольких тестов, и благодаря суффиксу `_test.go` компилируется только при запуске тестов.

### Трассировка требований публикации

| Требование | Реализация | Интеграционный тест |
| --- | --- | --- |
| Вернуть `403`, если `clientId != trip.DriverID` | [Проверка доменного правила](internal/domain/publish_trip.go#L30-L37), [преобразование в HTTP-ответ](internal/api/error_response.go) | [Сценарий запрета доступа](internal/api/move_trip_draft_to_published_test.go#L94-L105) |
| Вернуть `404`, если поездки нет | [Преобразование `pgx.ErrNoRows`](internal/repositories/trip.go#L143-L145), [преобразование в HTTP-ответ](internal/api/error_response.go) | [Сценарий отсутствующей поездки](internal/api/move_trip_draft_to_published_test.go#L107-L116) |
| Вернуть `409`, если поездка не в статусе `draft` | [Проверка доменного правила](internal/domain/publish_trip.go#L47-L54), [преобразование в HTTP-ответ](internal/api/error_response.go) | [Сценарий конфликта статусов](internal/api/move_trip_draft_to_published_test.go#L118-L136) |
| Вернуть `204`, если поездка уже в статусе `published` | [Проверка доменного правила](internal/domain/publish_trip.go#L39-L45), [преобразование в HTTP-ответ](internal/api/move_trip_draft_to_published.go) | [Сценарий ответа без содержимого](internal/api/move_trip_draft_to_published_test.go#L138-L165) |

## Проверки

```bash
make fmt
make lint
make test
make coverage
make build
make check
```

Интеграционные тесты запускают изолированный контейнер PostgreSQL 16, поэтому Docker должен быть запущен.
