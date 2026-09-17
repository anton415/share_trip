package service_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"job4j.ru/share-trip/internal/domain"
	"job4j.ru/share-trip/internal/service"
	"job4j.ru/share-trip/internal/service/mocks"
)

type startTripRepositoryStub struct {
	service.TripRepository
	trip     domain.Trip
	getCalls int
	updates  []domain.Trip
}

func (r *startTripRepositoryStub) GetForUpdateByID(
	_ context.Context,
	_ pgx.Tx,
	_ uuid.UUID,
) (domain.Trip, error) {
	r.getCalls++
	return r.trip, nil
}

func (r *startTripRepositoryStub) Update(
	_ context.Context,
	_ pgx.Tx,
	trip domain.Trip,
) (domain.Trip, error) {
	r.updates = append(r.updates, trip)
	return trip, nil
}

func TestTripServiceStartTrip(t *testing.T) {
	t.Parallel()

	command := service.StartTripCommand{TripID: uuid.New(), ClientID: uuid.New()}

	t.Run("allowed starts published trip", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		contracts := mocks.NewMockContractChecker(ctrl)
		contracts.EXPECT().
			CheckService(gomock.Any(), command.ClientID.String(), "trip_start").
			Return(service.CheckResult{Allowed: true, Reason: "service_allowed"}, nil)

		repo := &startTripRepositoryStub{trip: domain.Trip{
			ID:       command.TripID,
			DriverID: command.ClientID,
			Status:   domain.TripStatusPublished,
		}}
		txCalls := 0
		tripService := service.NewTripServiceForTest(repo, contracts, func(_ context.Context, block func(pgx.Tx) (*domain.Trip, error)) (*domain.Trip, error) {
			txCalls++
			return block(nil)
		})

		tripID, err := tripService.StartTrip(context.Background(), command)

		require.NoError(t, err)
		require.Equal(t, command.TripID, tripID)
		require.Equal(t, 1, txCalls)
		require.Equal(t, 1, repo.getCalls)
		require.Len(t, repo.updates, 1)
		require.Equal(t, domain.TripStatusStarted, repo.updates[0].Status)
	})

	t.Run("denied does not start transaction or save", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		contracts := mocks.NewMockContractChecker(ctrl)
		contracts.EXPECT().
			CheckService(gomock.Any(), command.ClientID.String(), "trip_start").
			Return(service.CheckResult{Allowed: false, Reason: "service_denied"}, nil)

		repo := &startTripRepositoryStub{}
		txCalls := 0
		tripService := service.NewTripServiceForTest(repo, contracts, func(_ context.Context, _ func(pgx.Tx) (*domain.Trip, error)) (*domain.Trip, error) {
			txCalls++
			return nil, nil
		})

		_, err := tripService.StartTrip(context.Background(), command)

		require.ErrorIs(t, err, domain.ErrForbidden)
		require.Zero(t, txCalls)
		require.Zero(t, repo.getCalls)
		require.Empty(t, repo.updates)
	})

	t.Run("client timeout does not start transaction or save", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		contracts := mocks.NewMockContractChecker(ctrl)
		contracts.EXPECT().
			CheckService(gomock.Any(), command.ClientID.String(), "trip_start").
			Return(service.CheckResult{}, context.DeadlineExceeded)

		repo := &startTripRepositoryStub{}
		txCalls := 0
		tripService := service.NewTripServiceForTest(repo, contracts, func(_ context.Context, _ func(pgx.Tx) (*domain.Trip, error)) (*domain.Trip, error) {
			txCalls++
			return nil, nil
		})

		_, err := tripService.StartTrip(context.Background(), command)

		require.ErrorIs(t, err, context.DeadlineExceeded)
		require.Zero(t, txCalls)
		require.Zero(t, repo.getCalls)
		require.Empty(t, repo.updates)
	})
}
