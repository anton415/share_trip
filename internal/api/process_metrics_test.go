package api_test

import (
	"net/http"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"

	"job4j.ru/share-trip/internal/api"
	observability "job4j.ru/share-trip/internal/observability/metrics"
)

func TestServer_ProcessMetrics(t *testing.T) {
	requireIntegration(t)
	t.Parallel()

	fixture := newTestFixture()
	createBefore := testutil.ToFloat64(
		fixture.metrics.TripCreateTotal.WithLabelValues(observability.ResultSuccess),
	)

	created := createDraftTrip(t, fixture.app)

	require.Equal(t, createBefore+1, testutil.ToFloat64(
		fixture.metrics.TripCreateTotal.WithLabelValues(observability.ResultSuccess),
	))

	publishBefore := testutil.ToFloat64(
		fixture.metrics.TripPublishTotal.WithLabelValues(observability.ResultSuccess),
	)

	resp := sendMoveTripDraftToPublished(t, fixture.app, api.MoveTripDraftToPublishedRequest{
		TripID:   created.ID.String(),
		ClientID: created.DriverID.String(),
	})
	defer closeResponseBody(t, resp.Body)

	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, publishBefore+1, testutil.ToFloat64(
		fixture.metrics.TripPublishTotal.WithLabelValues(observability.ResultSuccess),
	))
}
