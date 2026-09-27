-- +goose Up
-- +goose StatementBegin

ALTER TABLE outbox_event RENAME TO outbox_events;
ALTER TABLE outbox_events RENAME COLUMN event_name TO event_type;

ALTER TABLE outbox_events
    ADD COLUMN aggregate_type TEXT NOT NULL DEFAULT 'trip',
    ADD COLUMN status TEXT NOT NULL DEFAULT 'pending',
    ADD COLUMN attempts INT NOT NULL DEFAULT 0,
    ADD COLUMN last_error TEXT,
    ADD COLUMN sent_at TIMESTAMPTZ;

ALTER TABLE outbox_events ALTER COLUMN aggregate_type DROP DEFAULT;

-- Legacy events used a separate Kafka event_id that was not saved in outbox.
UPDATE outbox_events
SET status = 'failed',
    last_error = 'legacy event: original Kafka event_id and delivery status are unknown'
WHERE event_type = 'trip_published';

CREATE INDEX idx_outbox_events_pending
    ON outbox_events (created_at)
    WHERE status = 'pending';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX idx_outbox_events_pending;

ALTER TABLE outbox_events
    DROP COLUMN aggregate_type,
    DROP COLUMN status,
    DROP COLUMN attempts,
    DROP COLUMN last_error,
    DROP COLUMN sent_at;

ALTER TABLE outbox_events RENAME COLUMN event_type TO event_name;
ALTER TABLE outbox_events RENAME TO outbox_event;

-- +goose StatementEnd
