# Консистентность: saga, outbox и inbox

## Двойная запись

Без outbox ShareTrip сохраняет поездку в своей БД, затем отправляет событие в Kafka.
Это две отдельные записи: транзакция PostgreSQL не охватывает Kafka.

Например, поездка уже получила статус `published`, но перед отправкой события
сервис упал. Поездка опубликована, а Notification Service о ней не узнал.

```plantuml
@startuml
title Потеря события без outbox
participant ShareTrip
database "Trip DB" as TripDB
queue Kafka
participant "Notification Service" as Notification

ShareTrip -> TripDB: UPDATE trips SET status='published'
TripDB --> ShareTrip: committed
ShareTrip -> Kafka: publish TripPublished
Kafka --> ShareTrip: error / timeout
note right of ShareTrip
  Поездка опубликована,
  но события в Kafka нет.
end note
Kafka -[#red]-> Notification: нет сообщения
@enduml
```

Теперь `PublishTrip` сохраняет поездку и полный `TripPublished` в `outbox_events`
в одной транзакции. `OutboxPublisher` отправляет события отдельно от пользовательского запроса.

## Чтение, запись и retry

- **Чтение** получает информацию: например, проверка доступности `trip_start`
  в Contract Service. Повтор запроса не создаёт нового бизнес-действия.
- **Запись** меняет состояние: например, создание уведомления.
  Повтор без защиты может создать второе уведомление.

Retry — повтор после ошибки — помогает при временной недоступности.
Но если операция выполнена, а ответ потерялся, повтор может продублировать её.
Кроме того, после падения процесса нужно помнить, какое событие осталось отправить.
Для этого нужны outbox и inbox.

## Outbox: не потерять событие

Outbox находится в БД ShareTrip. Поездка и событие сохраняются в одной
транзакции: либо обе записи успешны, либо обе откатываются.
Publisher работает в отдельной goroutine: раз в секунду выбирает до 100 событий
`pending` по `created_at` с `FOR UPDATE SKIP LOCKED` и отправляет их в `trip.events`.
Выборка и запись результатов отправки выполняются в одной транзакции.

После подтверждения Kafka publisher устанавливает `sent` и `sent_at`.
При ошибке отправки увеличивает `attempts`, сохраняет `last_error` и оставляет `pending`.
Ошибка записи результата в БД откатывает транзакцию пачки.
Сбой после отправки, но до commit может привести к повторной доставке.
При повторах сохраняется тот же `event_id`, равный `outbox_events.id`.

Миграция сохраняет старые записи `trip_published` как `failed` с причиной в `last_error`.
Их прежний Kafka `event_id` и результат доставки неизвестны, поэтому автоматически они не отправляются.

```plantuml
@startuml
title Transactional outbox
participant "PublishTrip use case" as UseCase
database "Trip DB" as DB

UseCase -> DB: BEGIN
UseCase -> DB: UPDATE trips SET status='published'
UseCase -> DB: INSERT outbox_events (TripPublished)
UseCase -> DB: COMMIT
@enduml
```

## Inbox: не создать дубль

Inbox находится в БД Notification Service и хранит обработанные `event_id`.
Новый `event_id` и уведомление сохраняются в одной транзакции.
Уникальность `event_id` не позволяет обработать одно событие дважды.
Такая защита уже есть в проекте через `processed_events`.

Offset — позиция обработки сообщения в Kafka — подтверждается после commit.
При ошибке БД транзакция откатывается, offset не подтверждается.
Если сообщение придёт повторно после успешной обработки, сервис пропустит дубль.

```plantuml
@startuml
title Inbox transaction
queue "Kafka: trip.events" as Kafka
participant "Notification Consumer" as Notification
database "Notification DB" as DB

Kafka -> Notification: TripPublished(event_id)
Notification -> DB: BEGIN
Notification -> DB: INSERT processed_events(event_id)\nON CONFLICT DO NOTHING
alt Новое событие
  Notification -> DB: Создать уведомление
else event_id уже обработан
  note over Notification, DB: Пропустить создание
end
Notification -> DB: COMMIT
Notification -> Kafka: Подтвердить offset
@enduml
```

## Saga и компенсации

Saga связывает шаги в разных сервисах в один бизнес-процесс.
Если завершить процесс нельзя, выполненные шаги компенсируют новыми действиями.
Например, в ShareTrip можно отменить публикацию поездки. Если позже появятся
оплата и отправка сообщений, компенсациями могут быть возврат денег
и уведомление об отмене поездки.

Компенсация не откатывает чужую транзакцию: это отдельное бизнес-действие.
Временная ошибка создания уведомления сама по себе не требует отмены поездки —
достаточно повторить доставку. Outbox и inbox обеспечивают передачу событий,
а saga определяет шаги процесса и компенсации.

## Полный процесс ShareTrip -> outbox -> Kafka -> inbox -> Notification

`NotificationCreated` здесь означает сохранённое уведомление со статусом `created`.
Отдельное событие с таким именем сейчас в Kafka не отправляется.

[Полная PlantUML-диаграмма процесса](publish_trip.puml).

Уведомление может появиться с задержкой. После восстановления сервисов
и успешных повторов оно будет создано без дублей для того же `event_id`.
