package service

import (
	"github.com/jackc/pgx/v5/pgxpool"

	"job4j.ru/share-trip/internal/domain"
	observability "job4j.ru/share-trip/internal/observability/metrics"
)

func NewTripService(
	repo TripRepository,
	pool *pgxpool.Pool,
	metrics *observability.Metrics,
) *TripService {
	return &TripService{
		repo:        repo,
		pool:        pool,
		tripUsecase: domain.NewTripUsecase(repo),
		metrics:     metrics,
	}
}
