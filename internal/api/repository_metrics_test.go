package api_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"

	"job4j.ru/share-trip/internal/api"
	observability "job4j.ru/share-trip/internal/observability/metrics"
)

func TestServer_RepositoryMetrics(t *testing.T) {
	requireIntegration(t)
	t.Parallel()

	fixture := newTestFixture()
	createCounterBefore := repositoryCounterValue(
		fixture.metrics,
		observability.RepositoryOperationTripCreate,
		observability.ResultSuccess,
	)
	createDurationBefore := repositoryDurationCount(
		t,
		fixture.metrics,
		observability.RepositoryOperationTripCreate,
		observability.ResultSuccess,
	)

	created := createDraftTrip(t, fixture)

	require.Equal(t, createCounterBefore+1, repositoryCounterValue(
		fixture.metrics,
		observability.RepositoryOperationTripCreate,
		observability.ResultSuccess,
	))
	require.Equal(t, createDurationBefore+1, repositoryDurationCount(
		t,
		fixture.metrics,
		observability.RepositoryOperationTripCreate,
		observability.ResultSuccess,
	))

	publishOperations := []string{
		observability.RepositoryOperationTripGetForUpdateByID,
		observability.RepositoryOperationTripUpdate,
		observability.RepositoryOperationOutboxEventCreate,
	}
	publishBefore := make(map[string]float64, len(publishOperations))
	for _, operation := range publishOperations {
		publishBefore[operation] = repositoryCounterValue(
			fixture.metrics,
			operation,
			observability.ResultSuccess,
		)
	}

	publishResp := sendMoveTripDraftToPublished(t, fixture.app, api.MoveTripDraftToPublishedRequest{
		TripID: created.ID.String(),
	})
	require.Equal(t, http.StatusOK, publishResp.StatusCode)
	closeResponseBody(t, publishResp.Body)

	for _, operation := range publishOperations {
		require.Equal(t, publishBefore[operation]+1, repositoryCounterValue(
			fixture.metrics,
			operation,
			observability.ResultSuccess,
		))
	}

	getCounterBefore := repositoryCounterValue(
		fixture.metrics,
		observability.RepositoryOperationTripGetByID,
		observability.ResultSuccess,
	)

	getReq, err := http.NewRequest(http.MethodGet, "/trip/"+created.ID.String(), nil)
	require.NoError(t, err)
	getResp, err := fixture.app.Test(getReq, -1)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, getResp.StatusCode)
	closeResponseBody(t, getResp.Body)

	require.Equal(t, getCounterBefore+1, repositoryCounterValue(
		fixture.metrics,
		observability.RepositoryOperationTripGetByID,
		observability.ResultSuccess,
	))

	notFoundCounterBefore := repositoryCounterValue(
		fixture.metrics,
		observability.RepositoryOperationTripGetByID,
		observability.ResultNotFound,
	)

	notFoundReq, err := http.NewRequest(
		http.MethodGet,
		"/trip/"+uuid.NewString(),
		nil,
	)
	require.NoError(t, err)
	notFoundResp, err := fixture.app.Test(notFoundReq, -1)
	require.NoError(t, err)
	require.Equal(t, http.StatusNotFound, notFoundResp.StatusCode)
	closeResponseBody(t, notFoundResp.Body)

	require.Equal(t, notFoundCounterBefore+1, repositoryCounterValue(
		fixture.metrics,
		observability.RepositoryOperationTripGetByID,
		observability.ResultNotFound,
	))

	forUpdateNotFoundBefore := repositoryCounterValue(
		fixture.metrics,
		observability.RepositoryOperationTripGetForUpdateByID,
		observability.ResultNotFound,
	)

	notFoundPublishResp := sendMoveTripDraftToPublished(t, fixture.app, api.MoveTripDraftToPublishedRequest{
		TripID: uuid.NewString(),
	})
	require.Equal(t, http.StatusNotFound, notFoundPublishResp.StatusCode)
	closeResponseBody(t, notFoundPublishResp.Body)

	require.Equal(t, forUpdateNotFoundBefore+1, repositoryCounterValue(
		fixture.metrics,
		observability.RepositoryOperationTripGetForUpdateByID,
		observability.ResultNotFound,
	))
}

func repositoryCounterValue(
	appMetrics *observability.Metrics,
	operation string,
	result string,
) float64 {
	return testutil.ToFloat64(
		appMetrics.RepositoryQueryTotal.WithLabelValues(operation, result),
	)
}

func repositoryDurationCount(
	t *testing.T,
	appMetrics *observability.Metrics,
	operation string,
	result string,
) uint64 {
	t.Helper()

	observer := appMetrics.RepositoryQueryDuration.
		WithLabelValues(operation, result)
	metric, ok := observer.(prometheus.Metric)
	require.True(t, ok)

	var value dto.Metric
	require.NoError(t, metric.Write(&value))

	return value.GetHistogram().GetSampleCount()
}
