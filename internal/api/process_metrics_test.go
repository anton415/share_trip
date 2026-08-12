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
	createBefore := testutil.ToFloat64(
		testMetrics.TripCreateTotal.WithLabelValues(observability.ResultSuccess),
	)

	created := createDraftTrip(t)

	require.Equal(t, createBefore+1, testutil.ToFloat64(
		testMetrics.TripCreateTotal.WithLabelValues(observability.ResultSuccess),
	))

	publishBefore := testutil.ToFloat64(
		testMetrics.TripPublishTotal.WithLabelValues(observability.ResultSuccess),
	)

	resp := sendMoveTripDraftToPublished(t, api.MoveTripDraftToPublishedRequest{
		TripID:   created.ID,
		ClientID: created.DriverID,
	})
	defer closeResponseBody(t, resp.Body)

	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, publishBefore+1, testutil.ToFloat64(
		testMetrics.TripPublishTotal.WithLabelValues(observability.ResultSuccess),
	))
}
