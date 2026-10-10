package domain

import (
	"context"

	"github.com/jackc/pgx/v5"
)

type OutboxRepository interface {
	CreateOutboxEvent(ctx context.Context, tx pgx.Tx, event OutboxEvent) error
}
