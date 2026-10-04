package events

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"job4j.ru/share-trip/internal/domain"
	"job4j.ru/share-trip/internal/observability/metrics"
)

type outboxRepository interface {
	CountPending(ctx context.Context) (int, error)
	LockPending(ctx context.Context, tx pgx.Tx) ([]domain.OutboxEvent, error)
	MarkSent(ctx context.Context, tx pgx.Tx, eventID uuid.UUID) error
	MarkFailed(ctx context.Context, tx pgx.Tx, eventID uuid.UUID, publishErr error) error
}

type tripPublisher interface {
	PublishTripPublished(ctx context.Context, event domain.TripPublished) error
}

type OutboxPublisher struct {
	pool    *pgxpool.Pool
	outbox  outboxRepository
	kafka   tripPublisher
	logger  *slog.Logger
	metrics *metrics.Metrics
}

func NewOutboxPublisher(
	pool *pgxpool.Pool,
	outbox outboxRepository,
	kafka tripPublisher,
	logger *slog.Logger,
	appMetrics *metrics.Metrics,
) *OutboxPublisher {
	return &OutboxPublisher{pool: pool, outbox: outbox, kafka: kafka, logger: logger, metrics: appMetrics}
}

func (p *OutboxPublisher) Run(ctx context.Context) error {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := p.publishBatch(ctx); err != nil {
				p.logger.Error("publish outbox batch", "error", err)
			}
		}
	}
}

func (p *OutboxPublisher) publishBatch(ctx context.Context) error {
	publishErr := pgx.BeginFunc(ctx, p.pool, func(tx pgx.Tx) error {
		events, err := p.outbox.LockPending(ctx, tx)
		if err != nil {
			return err
		}

		for _, event := range events {
			var message domain.TripPublished
			err := json.Unmarshal(event.Payload, &message)
			if err != nil {
				err = fmt.Errorf("decode outbox event %s: %w", event.ID, err)
			} else {
				err = p.kafka.PublishTripPublished(ctx, message)
			}
			if err != nil {
				p.metrics.OutboxPublishTotal.WithLabelValues(metrics.ResultError).Inc()
				p.metrics.OutboxPublishFailed.Inc()
				if err := p.outbox.MarkFailed(ctx, tx, event.ID, err); err != nil {
					return err
				}
				continue
			}

			p.metrics.OutboxPublishTotal.WithLabelValues(metrics.ResultSuccess).Inc()
			if err := p.outbox.MarkSent(ctx, tx, event.ID); err != nil {
				return err
			}
		}

		return nil
	})
	pending, countErr := p.outbox.CountPending(ctx)
	if countErr == nil {
		p.metrics.OutboxPending.Set(float64(pending))
	}
	return errors.Join(publishErr, countErr)
}
