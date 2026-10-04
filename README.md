# ShareTrip

ShareTrip — серверный сервис для управления совместными поездками. Он реализует жизненный цикл поездки от создания черновика до публикации и разделяет доменные правила, слой хранения данных и HTTP-интерфейс.

## Возможности

- создание поездки в статусе `draft`;
- публикация поездки с проверкой прав доступа, текущего статуса и разрешения
  `trip_creation` в Contract Service (нужен активный договор с включённой услугой);
- получение поездки по идентификатору;
- аутентификация запросов к поездкам через Keycloak и проверка client-role;
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

Сервис читает переменные окружения через `config.Load()` до подключения к зависимостям.

| Переменная | Обязательная | Значение по умолчанию / назначение |
| --- | --- | --- |
| `HTTP_ADDR` | Нет | `:8080` |
| `DB_HOST` | Да | Адрес PostgreSQL |
| `DB_PORT` | Нет | `6544`; целое число от 1 до 65535 |
| `DB_USER` | Да | Пользователь PostgreSQL |
| `DB_PASSWORD` | Да | Пароль PostgreSQL; секрет |
| `DB_NAME` | Да | Имя базы данных |
| `DB_SSLMODE` | Нет | `disable` |
| `CONTRACT_SERVICE_BASE_URL` | Да | Адрес Contract Service |
| `CONTRACT_SERVICE_TIMEOUT_MS` | Нет | `2000`; положительное целое число миллисекунд, помещающееся в `time.Duration` |
| `CONTRACT_SERVICE_RETRY_COUNT` | Нет | `2`; целое число не меньше нуля |
| `KAFKA_BROKERS` | Да | Адреса Kafka-брокеров через запятую |
| `TRIP_EVENTS_TOPIC` | Нет | `trip.events` |
| `KEYCLOAK_ISSUER` | Да | URL realm в Keycloak |
| `KEYCLOAK_CLIENT_ID` | Нет | `sharetrip-api` |
| `KEYCLOAK_CLIENT_SECRET` | Да | Секрет клиента Keycloak |

При отсутствии обязательной переменной или неверном числовом значении сервис
завершается с ошибкой. Секреты и строка подключения с паролем не выводятся в
сообщениях об ошибках инициализации БД.

`DATABASE_URL` используется командами миграций. Приложение собирает строку
подключения из `DB_*`.

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

Задайте обязательные переменные для локального окружения из Docker Compose.
Замените `<keycloak-client-secret>` значением из `Clients → sharetrip-api → Credentials`.
Contract Service в этом примере должен быть доступен на порту `8082`.

```bash
export DB_HOST=localhost
export DB_USER=postgres
export DB_PASSWORD=password
export DB_NAME=sharetrip
export CONTRACT_SERVICE_BASE_URL=http://localhost:8082
export KAFKA_BROKERS=localhost:29092
export KEYCLOAK_ISSUER=http://localhost:8087/realms/sharetrip
export KEYCLOAK_CLIENT_SECRET='<keycloak-client-secret>'
make run
```

По умолчанию сервис доступен по адресу `http://localhost:8080`. В другом терминале проверьте его готовность:

```bash
make e2e
```

## Kafka

Запустите Kafka и Kafka UI и создайте topic `trip.events`:

```bash
docker compose -f deploy/docker-compose.yml up -d kafka kafka-ui
docker compose -f deploy/docker-compose.yml exec kafka /opt/kafka/bin/kafka-topics.sh \
  --bootstrap-server kafka:9092 --create --if-not-exists \
  --topic trip.events --partitions 1 --replication-factor 1
```

Kafka доступна приложениям на хосте по адресу `localhost:29092`, а контейнерам —
по адресу `kafka:9092`. Kafka UI: <http://localhost:8085>.
Адреса брокеров для ShareTrip обязательно задаются через `KAFKA_BROKERS`
(через запятую). Topic задаётся через `TRIP_EVENTS_TOPIC`, по умолчанию — `trip.events`.
После удаления контейнера Kafka topic нужно создать заново указанной командой.

## Kubernetes: конфигурация и секреты

Манифесты трёх сервисов находятся в [k8s/](k8s/). Каждый Deployment получает
настройки через `envFrom` из своей пары ConfigMap и Secret:

| Deployment | ConfigMap | Secret |
| --- | --- | --- |
| `sharetrip` | `sharetrip-config` | `sharetrip-secret` |
| `contract` | `contract-config` | `contract-secret` |
| `notification` | `notification-config` | `notification-secret` |

Связи показаны в [PlantUML-диаграмме](docs/configuration_secrets.puml).

Перед применением манифестов должен существовать namespace `sharetrip`, быть
доступны образы `sharetrip/<service>:local` и зависимости по адресам из ConfigMap и Secret.
В частности, `contract-service` должен разрешаться в адрес Contract Service.
В `sharetrip-config.yaml` заполните `KEYCLOAK_ISSUER`. Secret создавайте с рабочими
значениями по шаблонам `*-secret.example.yaml`; файлы с заполненными секретами
храните вне репозитория.

После изменения ConfigMap или Secret перезапустите соответствующий Deployment,
чтобы новые Pod-ы получили переменные окружения. Например:

```bash
kubectl rollout restart deployment/sharetrip -n sharetrip
```

## Метрики

После `make up` локальная инфраструктура мониторинга доступна по адресам:

| Компонент | Адрес | Назначение |
| --- | --- | --- |
| Prometheus | `http://localhost:9090` | Сбор и выполнение запросов к метрикам. |
| PostgreSQL Exporter | `http://localhost:9187/metrics` | Экспорт метрик PostgreSQL. |
| Grafana | `http://localhost:3000` | Визуализация метрик. Для первого запуска с чистым volume: `admin` / `admin`. |

Приложение запускается на хосте командой `make run`. Prometheus обращается к нему через `host.docker.internal:8080`, поэтому target `sharetrip` станет `UP` только после запуска приложения. Источник данных Prometheus в Grafana создаётся автоматически из файла конфигурации.

Notification Service экспортирует метрики на `:8081/metrics`; target `notification`
использует `host.docker.internal:8081` и станет `UP` после запуска этого сервиса.

Kafka Exporter собирает lag consumer group из Kafka; Prometheus обращается к нему
по внутреннему адресу `kafka-exporter:9308`.

Состояние сбора можно проверить на странице `http://localhost:9090/targets`. При запущенных сервисах пять target-ов — `prometheus`, `sharetrip`, `notification`, `kafka` и `postgres` — должны иметь состояние `UP`. Минимальные проверочные запросы в Prometheus:

```promql
up{job="sharetrip"}
up{job="notification"}
up{job="kafka"}
pg_up{job="postgres"}
go_goroutines{job="sharetrip"}
```

Метрики связей процесса:

| Метрика | Значение |
| --- | --- |
| `sharetrip_contract_request_total{result}` | Завершённые вызовы проверки прав; внутренние HTTP retries входят в один вызов. |
| `sharetrip_contract_request_duration_seconds{result}` | Длительность проверки прав вместе с retries. |
| `sharetrip_outbox_pending_total` | Gauge всех записей `pending`, обновляемый после каждого прохода publisher. При ошибке чтения сохраняется последнее значение. |
| `sharetrip_outbox_publish_total{result}` | Попытки публикации событий; успешная отправка учитывается даже при последующем откате статуса outbox. |
| `sharetrip_outbox_publish_failed_total` | Ошибки декодирования события или отправки в Kafka. |

Значения `result`: `success` и `error`. Корректный ответ Contract Service с
`allowed=false` — успешная проверка прав; отказ в доступе отражается в метрике публикации поездки.

Grafana автоматически загружает в папку `ShareTrip` четыре dashboard-а:

- `ShareTrip Runtime Go` — goroutines, память, GC, CPU и HTTP RPS;
- `ShareTrip PostgreSQL` — доступность, соединения, размер базы, commits и rollbacks;
- `ShareTrip Application` — HTTP RPS и p95, результаты операций с поездками и p95 репозитория;
- [Trip publication](http://localhost:3000/d/sharetrip-trip-publication) — rate и доля ошибок публикации поездок, p95 HTTP-команды и вызова Contract Service, outbox backlog и ошибки его публикации, Kafka lag, inbox duplicates и доля ошибок создания уведомлений.

Dashboard `Trip publication` использует topic `trip.events` и consumer group
`notification-service`. Для lag используется `kafka_consumergroup_lag` из
[Kafka Exporter](https://github.com/danielqsj/kafka_exporter#consumer-groups).
До появления consumer group и её offsets панель lag может показывать `No data`.
`notification_send_total` измеряет сохранение уведомления по `TripPublished` в БД.
Панели ошибок и дублей показывают прирост за последние 5 минут; доли ошибок — от 0 до 100%.

Для уже запущенного стека примените изменения мониторинга:

```bash
docker compose -f deploy/docker-compose.yml up -d kafka-exporter grafana
docker compose -f deploy/docker-compose.yml kill -s SIGHUP prometheus
```

Dashboard-ы хранятся как код и недоступны для сохранения изменений через UI. Панели бизнес-процессов и репозитория появляются после соответствующих HTTP-запросов и двух циклов сбора Prometheus. При отсутствии операций в выбранном временном диапазоне p95 не равен нулю — данных для вычисления квантиля ещё нет.

Остановите локальную инфраструктуру:

```bash
make down
```

## Как расследовать: уведомление не пришло

Связь этапов показана в [PlantUML-диаграмме публикации](docs/publish_trip.puml).
Начните с [dashboard Trip publication](http://localhost:3000/d/sharetrip-trip-publication),
затем найдите конкретный запрос и событие по идентификаторам.

1. Найдите ответ на `POST /trip/publish`: `tripId`, `X-Request-ID`,
   `X-Correlation-ID` и, если обработчик был вызван, `trace-id`.
   `200` означает новую публикацию; `204` — поездка уже опубликована, новое событие не создаётся.
   При ошибке сначала разберите HTTP-ответ.
2. Найдите связанные записи ShareTrip в `logs/app.log` или Loki по `request_id`
   и `correlation_id`; trace можно открыть в Jaeger по `trace-id`.
   Middleware добавляет эти поля в контекст логгера, но отдельного access log
   каждого запроса сейчас нет. Отсутствие строки не доказывает отсутствие вызова.
   В локальной конфигурации Alloy собирает только ShareTrip; логи Contract и
   Notification смотрите в stdout соответствующих процессов.
3. В Contract найдите `service=contract`, `operation=CheckPermission` с тем же
   `request_id`/`correlation_id` и `trip_id`. Проверьте `result=allowed|denied|error`.
   Для публикации нужна услуга `trip_creation`; отказ или ошибка проверки
   прерывают публикацию до транзакции поездки и outbox.
4. В БД ShareTrip проверьте поездку и `outbox_events` запросами ниже.
   `outbox_events.id` — это `event_id`; `payload.correlation_id` связывает событие
   с процессом, `payload.causation_id` — с исходным `request_id`.
   Если записи нет, проверьте результат транзакции публикации.
   Для `pending` проверьте `attempts`, `last_error` и работу publisher.
   `sent` вместе с `sent_at` означает подтверждённую отправку в Kafka;
   `last_error` может оставаться от предыдущей попытки. Даже `pending` может уже
   попасть в Kafka, если после отправки не удалось сохранить статус.
5. В Kafka UI найдите событие в topic `trip.events` по `event_id` из headers
   или JSON. Сверьте `event_type`, `correlation_id`, `causation_id`, `traceparent`.
   На dashboard проверьте lag группы `notification-service`: рост показывает
   отставание, а `No data` не означает нулевой lag. Если настройки topic/group
   изменены, используйте значения `TRIP_EVENTS_TOPIC` и `KAFKA_GROUP_ID`.
6. В stdout Notification найдите `service=notification`,
   `operation=ConsumeTripPublished`, нужный `event_id` и `correlation_id`.
   Сопоставьте `causation_id` с исходным запросом, `trace_id` — с его trace.
   Проверьте `result` и `error`: ошибки декодирования, обработки или подтверждения
   offset останавливают consumer. `success` записывается после подтверждения offset,
   в том числе при обработке дубля.
7. В БД Notification проверьте inbox `processed_events` и уведомление по тому же
   `event_id`. Они сохраняются одной транзакцией; повторное событие увеличивает
   `notification_inbox_duplicate_total`, но не создаёт новое уведомление.
   Рост `notification_send_total{result="error"}` указывает на ошибки сохранения.
   Уведомление со статусом `created` — конечный результат текущей версии:
   отправки во внешний канал и подтверждения доставки от провайдера пока нет.

Запросы для БД ShareTrip; замените `<trip-id>` идентификатором поездки:

```sql
SELECT id, status FROM trips WHERE id = '<trip-id>'::uuid;

SELECT id AS event_id, status, attempts, last_error, sent_at,
       payload->>'correlation_id' AS correlation_id,
       payload->>'causation_id' AS causation_id,
       payload->>'traceparent' AS traceparent
FROM outbox_events
WHERE aggregate_id = '<trip-id>'::uuid AND event_type = 'TripPublished';
```

Запросы для БД Notification; замените `<event-id>` значением из outbox:

```sql
SELECT event_id, processed_at FROM processed_events
WHERE event_id = '<event-id>'::uuid;

SELECT id, status, created_at FROM notifications
WHERE type = 'trip_published' AND payload->>'event_id' = '<event-id>';
```

## HTTP-интерфейс

| Метод | Путь | Описание |
| --- | --- | --- |
| `GET` | `/ready` | Проверяет готовность сервиса и соединение с PostgreSQL. |
| `GET` | `/metrics` | Возвращает runtime- и прикладные метрики в формате Prometheus. |
| `POST` | `/trip/create` | Создаёт поездку в статусе `draft`. |
| `POST` | `/trip/publish` | Публикует черновик поездки. |
| `GET` | `/trip/:id` | Возвращает поездку по идентификатору. |

Все маршруты `/trip` требуют заголовок `X-Refresh-Token`. Маршруты создания и
получения поездки дополнительно требуют роль `client` клиента
`KEYCLOAK_CLIENT_ID`. Идентификатор водителя берётся из JWT claim `sub`, поэтому
передавать `driverId` или `clientId` в теле запроса не нужно.

## Структура `internal`

Основу приложения составляют четыре пакета с разными зонами ответственности:

| Пакет | Назначение |
| --- | --- |
| [`api`](internal/api) | HTTP-ручки, транспортные DTO, маршруты и преобразование ошибок в HTTP-ответы. |
| [`domain`](internal/domain) | Доменные сущности, ошибки и бизнес-правила. |
| [`repository`](internal/repository) | Реализация хранения поездок и исходящих событий в PostgreSQL. |
| [`service`](internal/service) | Оркестрация прикладных сценариев и управление транзакциями. |

Технические пакеты вынесены отдельно: `config` загружает конфигурацию, `db` создаёт подключение к PostgreSQL, `middleware` содержит сквозные HTTP-компоненты, а `observability` объединяет настройку логирования, контекст логов и метрики.

## Структура HTTP-слоя

В пакете `internal/api` каждый HTTP-сценарий находится в отдельном файле вместе со своими DTO и обработчиком. Остальные production-функции и методы HTTP-границы также разнесены по отдельным файлам, имя каждого файла отражает выполняемое действие. Сквозные HTTP-компоненты не смешиваются с ручками и находятся в пакете `internal/middleware`.

| Файл | Назначение |
| --- | --- |
| [`create_trip.go`](internal/api/create_trip.go) | DTO и обработчик создания поездки. |
| [`move_trip_draft_to_published.go`](internal/api/move_trip_draft_to_published.go) | DTO и обработчик перевода поездки из `draft` в `published`. |
| [`get_trip_by_id.go`](internal/api/get_trip_by_id.go) | DTO и обработчик получения поездки по идентификатору. |
| [`ready.go`](internal/api/ready.go) | Обработчик проверки готовности сервиса. |
| [`metrics_handler.go`](internal/api/metrics_handler.go) | Адаптер обработчика метрик Prometheus для Fiber. |
| [`server.go`](internal/api/server.go) | Тип сервера и его зависимости. |
| [`new_server.go`](internal/api/new_server.go) | Создание сервера. |
| [`register_routes.go`](internal/api/register_routes.go) | Регистрация HTTP-маршрутов. |
| [`write_fiber_service_error.go`](internal/api/write_fiber_service_error.go) | Преобразование ошибок сервиса в HTTP-ответы. |
| [`new_create_trip_response.go`](internal/api/new_create_trip_response.go) | Преобразование созданной поездки в DTO ответа. |
| [`new_get_trip_by_id_response.go`](internal/api/new_get_trip_by_id_response.go) | Преобразование найденной поездки в DTO ответа. |
| [`integration_setup_test.go`](internal/api/integration_setup_test.go) | Подготовка PostgreSQL и приложения для интеграционных тестов. |
| [`helpers_test.go`](internal/api/helpers_test.go) | Общая подготовка тестовых данных и функции проверок. |

| Middleware | Назначение |
| --- | --- |
| [`correlation.go`](internal/middleware/correlation.go) | Добавляет идентификатор запроса и связанный с ним логгер в контекст. |
| [`http_metrics.go`](internal/middleware/http_metrics.go) | Измеряет количество и длительность HTTP-запросов. |

Go компилирует все файлы `.go`, не относящиеся к тестам и объявляющие `package api`, как единый пакет. Поэтому обработчики могут вызывать закрытые преобразователи ответов и общий mapper ошибок, хотя каждая production-функция находится в отдельном файле. DTO остаются рядом с соответствующими ручками: они описывают контракт конкретного endpoint и не нарушают правило одной функции на файл. Тестовые helpers являются инфраструктурой тестов и не дробятся механически.

### Трассировка требований публикации

| Требование | Реализация | Интеграционный тест |
| --- | --- | --- |
| Вернуть `403`, если `clientId != trip.DriverID` | [Проверка доменного правила](internal/domain/publish_trip.go), [преобразование в HTTP-ответ](internal/api/write_fiber_service_error.go) | [Сценарий запрета доступа](internal/api/move_trip_draft_to_published_test.go) |
| Вернуть `404`, если поездки нет | [Преобразование `pgx.ErrNoRows`](internal/repository/trip.go), [преобразование в HTTP-ответ](internal/api/write_fiber_service_error.go) | [Сценарий отсутствующей поездки](internal/api/move_trip_draft_to_published_test.go) |
| Вернуть `409`, если поездка не в статусе `draft` | [Проверка доменного правила](internal/domain/publish_trip.go), [преобразование в HTTP-ответ](internal/api/write_fiber_service_error.go) | [Сценарий конфликта статусов](internal/api/move_trip_draft_to_published_test.go) |
| Вернуть `204`, если поездка уже в статусе `published` | [Проверка доменного правила](internal/domain/publish_trip.go), [преобразование в HTTP-ответ](internal/api/move_trip_draft_to_published.go) | [Сценарий ответа без содержимого](internal/api/move_trip_draft_to_published_test.go) |

## Проверки

```bash
make fmt
make lint
make test
make coverage
make build
make check
```

Интеграционные тесты запускают изолированный контейнер PostgreSQL 16. Если Docker
недоступен, эти тесты корректно пропускаются; остальные тесты продолжают выполняться.
