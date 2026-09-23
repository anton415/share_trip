package service

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"job4j.ru/share-trip/internal/domain"
	observability "job4j.ru/share-trip/internal/observability/metrics"
)

func NewTripService(
	repo TripRepository,
	pool *pgxpool.Pool,
	metrics *observability.Metrics,
	contracts ContractChecker,
	publisher TripPublisher,
) *TripService {
	return &TripService{
		contracts: contracts,
		publisher: publisher,
		repo:      repo,
		pool:      pool,
		startTripTx: func(ctx context.Context, block func(pgx.Tx) (*domain.Trip, error)) (*domain.Trip, error) {
			return tx(ctx, pool, block)
		},
		tripUsecase: domain.NewTripUsecase(repo),
		metrics:     metrics,
	}
}
