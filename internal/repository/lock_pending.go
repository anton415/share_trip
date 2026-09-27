package repo

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"job4j.ru/share-trip/internal/domain"
)

// LockPending keeps the selected events locked until tx commits or rolls back.
func (r *PostgresTripRepository) LockPending(ctx context.Context, tx pgx.Tx) ([]domain.OutboxEvent, error) {
	rows, err := tx.Query(ctx, `
		SELECT id, aggregate_type, aggregate_id, event_type, payload,
		       status, attempts, last_error, created_at, sent_at
		FROM outbox_events
		WHERE status = 'pending'
		ORDER BY created_at
		LIMIT 100
		FOR UPDATE SKIP LOCKED
	`)
	if err != nil {
		return nil, fmt.Errorf("select pending outbox events: %w", err)
	}

	events, err := pgx.CollectRows(rows, pgx.RowToStructByPos[domain.OutboxEvent])
	if err != nil {
		return nil, fmt.Errorf("read pending outbox events: %w", err)
	}

	return events, nil
}
