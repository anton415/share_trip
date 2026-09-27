package repo

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (r *PostgresTripRepository) MarkSent(ctx context.Context, tx pgx.Tx, eventID uuid.UUID) error {
	_, err := tx.Exec(ctx, `
		UPDATE outbox_events
		SET status = 'sent', sent_at = now()
		WHERE id = $1
	`, eventID)
	if err != nil {
		return fmt.Errorf("mark outbox event sent: %w", err)
	}

	return nil
}

func (r *PostgresTripRepository) MarkFailed(
	ctx context.Context,
	tx pgx.Tx,
	eventID uuid.UUID,
	publishErr error,
) error {
	_, err := tx.Exec(ctx, `
		UPDATE outbox_events
		SET attempts = attempts + 1, last_error = $2
		WHERE id = $1
	`, eventID, publishErr.Error())
	if err != nil {
		return fmt.Errorf("record outbox event failure: %w", err)
	}

	return nil
}
