package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"job4j.ru/share-trip/internal/domain"
	observability "job4j.ru/share-trip/internal/observability/metrics"
)

type TripRepository interface {
	domain.TripRepository

	GetTripByID(ctx context.Context, id uuid.UUID) (domain.Trip, error)
}

type TripService struct {
	repo        TripRepository
	pool        *pgxpool.Pool
	tripUsecase *domain.TripUsecase
	metrics     *observability.Metrics
}
