package service

import (
	"context"

	"github.com/google/uuid"

	"job4j.ru/share-trip/internal/domain"
)

func (s *TripService) GetTripByID(ctx context.Context, id uuid.UUID) (domain.Trip, error) {
	return s.repo.GetTripByID(ctx, id)
}
