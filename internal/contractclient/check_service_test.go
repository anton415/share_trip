package contractclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/propagation"

	"job4j.ru/share-trip/internal/observability/logctx"
	"job4j.ru/share-trip/internal/observability/metrics"
)

func TestClientCheckService(t *testing.T) {
	t.Parallel()

	const traceparent = "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"
	headers := make(chan http.Header, 1)
	contractService := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		headers <- r.Header.Clone()
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/contracts/check-service", r.URL.Path)

		var request struct {
			ClientID    string `json:"client_id"`
			ServiceCode string `json:"service_code"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
		require.Equal(t, "company-123", request.ClientID)
		require.Equal(t, "trip_start", request.ServiceCode)

		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{"allowed":true,"reason":"service_allowed"}`))
		require.NoError(t, err)
	}))
	defer contractService.Close()

	ctx := logctx.WithRequestID(context.Background(), "req-123")
	ctx = logctx.WithCorrelationID(ctx, "pub-777")
	ctx = logctx.WithTripID(ctx, "trip-42")
	ctx = propagation.TraceContext{}.Extract(ctx, propagation.MapCarrier{"traceparent": traceparent})
	result, err := New(contractService.URL, time.Second, 0, metrics.New(prometheus.NewRegistry())).
		CheckService(ctx, "company-123", "trip_start")

	require.NoError(t, err)
	received := <-headers
	require.Equal(t, "req-123", received.Get("X-Request-ID"))
	require.Equal(t, "pub-777", received.Get("X-Correlation-ID"))
	require.Equal(t, "trip-42", received.Get("X-Trip-ID"))
	require.Equal(t, traceparent, received.Get("traceparent"))
	require.True(t, result.Allowed)
	require.Equal(t, "service_allowed", result.Reason)
}

func TestClientCheckServiceMetrics(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name, body, result string
		status             int
	}{
		{"allowed", `{"allowed":true,"reason":"service_allowed"}`, "success", http.StatusOK},
		{"denied", `{"allowed":false,"reason":"service_not_allowed"}`, "success", http.StatusOK},
		{"invalid response", `{`, "error", http.StatusOK},
		{"unavailable", `{}`, "error", http.StatusServiceUnavailable},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.status)
				if _, err := w.Write([]byte(tt.body)); err != nil {
					t.Error(err)
				}
			}))
			defer server.Close()
			appMetrics := metrics.New(prometheus.NewRegistry())
			_, err := New(server.URL, time.Second, 0, appMetrics).CheckService(t.Context(), "client", "trip_creation")
			require.Equal(t, tt.result == "error", err != nil)
			require.Equal(t, float64(1), testutil.ToFloat64(appMetrics.ContractRequestTotal.WithLabelValues(tt.result)))
			var sample dto.Metric
			require.NoError(t, appMetrics.ContractRequestDuration.WithLabelValues(tt.result).(prometheus.Metric).Write(&sample))
			require.Equal(t, uint64(1), sample.GetHistogram().GetSampleCount())
			require.Positive(t, sample.GetHistogram().GetSampleSum())
		})
	}
}
