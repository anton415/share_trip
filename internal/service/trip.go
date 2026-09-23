package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"job4j.ru/share-trip/internal/domain"
	"job4j.ru/share-trip/internal/events"
	observability "job4j.ru/share-trip/internal/observability/metrics"
)

type TripRepository interface {
	domain.TripRepository

	GetTripByID(ctx context.Context, id uuid.UUID) (domain.Trip, error)
}

type TripPublisher interface {
	PublishTripPublished(ctx context.Context, event events.TripPublished) error
}

type TripService struct {
	contracts   ContractChecker
	publisher   TripPublisher
	repo        TripRepository
	pool        *pgxpool.Pool
	startTripTx startTripTransaction
	tripUsecase *domain.TripUsecase
	metrics     *observability.Metrics
}
