package api_test

import (
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestServer_Metrics(t *testing.T) {
	requireIntegration(t)
	t.Parallel()

	fixture := newTestFixture()
	fixture.metrics.TripCreateTotal.WithLabelValues("success").Inc()

	req, err := http.NewRequest(http.MethodGet, "/metrics", nil)
	require.NoError(t, err)

	resp, err := fixture.app.Test(req, -1)
	require.NoError(t, err)
	defer closeResponseBody(t, resp.Body)

	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Contains(
		t,
		string(body),
		`sharetrip_trip_create_total{result="success"}`,
	)
}
