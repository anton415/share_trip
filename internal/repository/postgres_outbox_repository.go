package repo

import (
	"github.com/jackc/pgx/v5/pgxpool"

	observability "job4j.ru/share-trip/internal/observability/metrics"
)

type PostgresOutboxRepository struct {
	pool    *pgxpool.Pool
	metrics *observability.Metrics
}

func NewPostgresOutboxRepository(
	pool *pgxpool.Pool,
	metrics *observability.Metrics,
) *PostgresOutboxRepository {
	return &PostgresOutboxRepository{
		pool:    pool,
		metrics: metrics,
	}
}
