package api_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"

	"job4j.ru/share-trip/internal/api"
	observability "job4j.ru/share-trip/internal/observability/metrics"
)

func TestHTTPMetricsMiddleware(t *testing.T) {
	t.Run("uses route template", func(t *testing.T) {
		appMetrics := observability.New(prometheus.NewRegistry())
		app := fiber.New()
		app.Use(api.NewHTTPMetricsMiddleware(appMetrics))
		app.Get("/trip/:id", func(c *fiber.Ctx) error {
			return c.SendStatus(fiber.StatusCreated)
		})

		req, err := http.NewRequest(http.MethodGet, "/trip/123", nil)
		require.NoError(t, err)

		resp, err := app.Test(req)
		require.NoError(t, err)
		require.NoError(t, resp.Body.Close())
		require.Equal(t, fiber.StatusCreated, resp.StatusCode)
		require.Equal(t, float64(1), testutil.ToFloat64(
			appMetrics.HTTPRequestTotal.WithLabelValues(
				http.MethodGet,
				"/trip/:id",
				"201",
			),
		))
	})

	t.Run("records Fiber error status", func(t *testing.T) {
		appMetrics := observability.New(prometheus.NewRegistry())
		app := fiber.New()
		app.Use(api.NewHTTPMetricsMiddleware(appMetrics))
		app.Get("/error", func(*fiber.Ctx) error {
			return fiber.ErrTeapot
		})

		req, err := http.NewRequest(http.MethodGet, "/error", nil)
		require.NoError(t, err)

		resp, err := app.Test(req)
		require.NoError(t, err)
		require.NoError(t, resp.Body.Close())
		require.Equal(t, fiber.StatusTeapot, resp.StatusCode)
		require.Equal(t, float64(1), testutil.ToFloat64(
			appMetrics.HTTPRequestTotal.WithLabelValues(
				http.MethodGet,
				"/error",
				"418",
			),
		))
	})

	t.Run("uses bounded label for unmatched route", func(t *testing.T) {
		appMetrics := observability.New(prometheus.NewRegistry())
		app := fiber.New()
		app.Use(api.NewHTTPMetricsMiddleware(appMetrics))

		req, err := http.NewRequest(
			http.MethodGet,
			"/missing/550e8400-e29b-41d4-a716-446655440000",
			nil,
		)
		require.NoError(t, err)

		resp, err := app.Test(req)
		require.NoError(t, err)
		require.NoError(t, resp.Body.Close())
		require.Equal(t, fiber.StatusNotFound, resp.StatusCode)
		require.Equal(t, float64(1), testutil.ToFloat64(
			appMetrics.HTTPRequestTotal.WithLabelValues(
				http.MethodGet,
				"unmatched",
				"404",
			),
		))
	})

	t.Run("copies method label before Fiber reuses context", func(t *testing.T) {
		appMetrics := observability.New(prometheus.NewRegistry())
		app := fiber.New()
		app.Use(api.NewHTTPMetricsMiddleware(appMetrics))
		app.Post("/trip/create", func(c *fiber.Ctx) error {
			return c.SendStatus(fiber.StatusCreated)
		})
		app.Get("/ready", func(c *fiber.Ctx) error {
			return c.SendStatus(fiber.StatusOK)
		})

		postReq, err := http.NewRequest(http.MethodPost, "/trip/create", nil)
		require.NoError(t, err)
		postResp, err := app.Test(postReq)
		require.NoError(t, err)
		require.NoError(t, postResp.Body.Close())

		getReq, err := http.NewRequest(http.MethodGet, "/ready", nil)
		require.NoError(t, err)
		getResp, err := app.Test(getReq)
		require.NoError(t, err)
		require.NoError(t, getResp.Body.Close())

		require.NoError(t, testutil.CollectAndCompare(
			appMetrics.HTTPRequestTotal,
			strings.NewReader(`
# HELP sharetrip_http_requests_total Total number of HTTP requests
# TYPE sharetrip_http_requests_total counter
sharetrip_http_requests_total{method="GET",path="/ready",status="200"} 1
sharetrip_http_requests_total{method="POST",path="/trip/create",status="201"} 1
`),
			"sharetrip_http_requests_total",
		))
	})
}
