package service_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"job4j.ru/share-trip/internal/domain"
	"job4j.ru/share-trip/internal/observability/logctx"
	"job4j.ru/share-trip/internal/observability/metrics"
	"job4j.ru/share-trip/internal/service"
	"job4j.ru/share-trip/internal/service/mocks"
)

func TestPublishTripChecksPermissionBeforeTransaction(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name    string
		err     error
		wantErr error
	}{
		{name: "denied", wantErr: domain.ErrForbidden},
		{name: "timeout", err: context.DeadlineExceeded, wantErr: context.DeadlineExceeded},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			command := service.PublishTripCommand{TripID: uuid.New(), ClientID: uuid.New()}
			contracts := mocks.NewMockContractChecker(gomock.NewController(t))
			contracts.EXPECT().CheckService(gomock.Any(), command.ClientID.String(), "trip_creation").
				DoAndReturn(func(ctx context.Context, _, _ string) (service.CheckResult, error) {
					require.Equal(t, command.TripID.String(), logctx.TripID(ctx))
					require.Equal(t, "req-123", logctx.RequestID(ctx))
					require.Equal(t, "pub-777", logctx.CorrelationID(ctx))
					return service.CheckResult{}, tt.err
				})
			// A rejected check must not access the database or repository.
			tripService := service.NewTripService(nil, nil, metrics.New(prometheus.NewRegistry()), contracts)
			ctx := logctx.WithRequestID(context.Background(), "req-123")
			ctx = logctx.WithCorrelationID(ctx, "pub-777")
			id, err := tripService.PublishTrip(ctx, command)
			require.ErrorIs(t, err, tt.wantErr)
			require.Equal(t, uuid.Nil, id)
		})
	}
}
