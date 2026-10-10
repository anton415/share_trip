package repo

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"job4j.ru/share-trip/internal/domain"
	observability "job4j.ru/share-trip/internal/observability/metrics"
)

func (r *PostgresOutboxRepository) CreateOutboxEvent(
	ctx context.Context,
	tx pgx.Tx,
	event domain.OutboxEvent,
) error {
	ctx, span := otel.Tracer("OutboxRepository").
		Start(ctx, "OutboxRepository.CreateOutboxEvent")
	defer span.End()

	span.SetAttributes(
		attribute.String("trip_id", event.AggregateID.String()),
		attribute.String("event_type", event.EventType),
	)

	started := time.Now()
	result := observability.ResultSuccess
	defer func() {
		r.metrics.RepositoryQueryTotal.
			WithLabelValues(observability.RepositoryOperationOutboxEventCreate, result).
			Inc()
		r.metrics.RepositoryQueryDuration.
			WithLabelValues(observability.RepositoryOperationOutboxEventCreate, result).
			Observe(time.Since(started).Seconds())
	}()

	_, err := tx.Exec(ctx, `
		INSERT INTO outbox_events (
			id,
			aggregate_type,
			aggregate_id,
			event_type,
			payload
		)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (id) DO NOTHING
	`,
		event.ID,
		event.AggregateType,
		event.AggregateID,
		event.EventType,
		event.Payload,
	)
	if err != nil {
		result = observability.ResultInternalError
		return fmt.Errorf("insert outbox event: %w", err)
	}

	return nil
}
