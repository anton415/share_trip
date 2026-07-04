package service

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func tx[T any](
	ctx context.Context,
	pool *pgxpool.Pool,
	block func(pgx.Tx) (*T, error),
) (*T, error) {
	txBegin, err := pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer txBegin.Rollback(ctx)

	result, err := block(txBegin)
	if err != nil {
		return nil, fmt.Errorf("transaction block: %w", err)
	}

	if err := txBegin.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return result, nil
}
