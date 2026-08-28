package repo

import (
	"github.com/jackc/pgx/v5/pgxpool"

	observability "job4j.ru/share-trip/internal/observability/metrics"
)

type PostgresTripRepository struct {
	pool    *pgxpool.Pool
	metrics *observability.Metrics
}

func NewPostgresTripRepository(
	pool *pgxpool.Pool,
	metrics *observability.Metrics,
) *PostgresTripRepository {
	return &PostgresTripRepository{
		pool:    pool,
		metrics: metrics,
	}
}
