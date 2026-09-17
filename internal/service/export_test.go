package service

import (
	"context"

	"github.com/jackc/pgx/v5"
	"job4j.ru/share-trip/internal/domain"
)

func NewTripServiceForTest(
	repo TripRepository,
	contracts ContractChecker,
	startTripTx func(context.Context, func(pgx.Tx) (*domain.Trip, error)) (*domain.Trip, error),
) *TripService {
	return &TripService{
		contracts:   contracts,
		repo:        repo,
		startTripTx: startTripTx,
	}
}
