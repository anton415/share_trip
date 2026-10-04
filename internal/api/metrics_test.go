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
	fixture.metrics.ContractRequestTotal.WithLabelValues("success").Inc()
	fixture.metrics.ContractRequestDuration.WithLabelValues("success").Observe(0.25)
	fixture.metrics.OutboxPending.Set(7)
	fixture.metrics.OutboxPublishTotal.WithLabelValues("error").Inc()
	fixture.metrics.OutboxPublishFailed.Inc()

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
	for _, line := range []string{
		`sharetrip_contract_request_total{result="success"} 1`,
		`sharetrip_contract_request_duration_seconds_count{result="success"} 1`,
		`sharetrip_contract_request_duration_seconds_sum{result="success"} 0.25`,
		`sharetrip_outbox_pending_total 7`,
		`sharetrip_outbox_publish_total{result="error"} 1`,
		`sharetrip_outbox_publish_failed_total 1`,
	} {
		require.Contains(t, string(body), line)
	}
}
