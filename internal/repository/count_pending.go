package repo

import (
	"context"
	"fmt"
)

func (r *PostgresOutboxRepository) CountPending(ctx context.Context) (int, error) {
	var count int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM outbox_events WHERE status = 'pending'`).Scan(&count); err != nil {
		return 0, fmt.Errorf("count pending outbox events: %w", err)
	}
	return count, nil
}
