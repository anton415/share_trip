package middleware_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"testing"

	"github.com/gofiber/contrib/otelfiber/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"

	"job4j.ru/share-trip/internal/middleware"
	"job4j.ru/share-trip/internal/observability/logctx"
)

func TestCorrelation(t *testing.T) {
	t.Parallel()

	const incomingTraceID = "4bf92f3577b34da6a3ce929d0e0e4736"
	tests := []struct {
		name          string
		requestID     string
		correlationID string
		traceparent   string
	}{
		{name: "generates identifiers when headers are absent"},
		{name: "uses request ID as correlation ID", requestID: "req-123"},
		{name: "preserves correlation ID with generated request ID", correlationID: "pub-777"},
		{
			name:          "preserves incoming identifiers and continues trace",
			requestID:     "req-123",
			correlationID: "pub-777",
			traceparent:   "00-" + incomingTraceID + "-00f067aa0ba902b7-01",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var logs bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&logs, nil))
			provider := sdktrace.NewTracerProvider()
			t.Cleanup(func() {
				require.NoError(t, provider.Shutdown(t.Context()))
			})

			app := fiber.New()
			app.Use(otelfiber.Middleware(
				otelfiber.WithTracerProvider(provider),
				otelfiber.WithPropagators(propagation.TraceContext{}),
			))
			app.Use(middleware.Correlation(logger))
			app.Get("/test", func(c *fiber.Ctx) error {
				ctx := c.UserContext()
				logctx.Logger(ctx).Info("handle request")
				return c.JSON(fiber.Map{
					"request_id":     logctx.RequestID(ctx),
					"correlation_id": logctx.CorrelationID(ctx),
					"trace_id":       trace.SpanContextFromContext(ctx).TraceID().String(),
				})
			})

			req, err := http.NewRequest(http.MethodGet, "/test", nil)
			require.NoError(t, err)
			req.Header.Set(middleware.RequestIDHeader, tt.requestID)
			req.Header.Set(middleware.CorrelationIDHeader, tt.correlationID)
			req.Header.Set("traceparent", tt.traceparent)

			resp, err := app.Test(req)
			require.NoError(t, err)
			defer func() { require.NoError(t, resp.Body.Close()) }()
			require.Equal(t, http.StatusOK, resp.StatusCode)

			var identifiers map[string]string
			require.NoError(t, json.NewDecoder(resp.Body).Decode(&identifiers))
			requestID := identifiers["request_id"]
			if tt.requestID == "" {
				_, err := uuid.Parse(requestID)
				require.NoError(t, err)
			} else {
				require.Equal(t, tt.requestID, requestID)
			}
			if tt.correlationID == "" {
				require.Equal(t, requestID, identifiers["correlation_id"])
			} else {
				require.Equal(t, tt.correlationID, identifiers["correlation_id"])
			}
			require.Equal(t, requestID, resp.Header.Get(middleware.RequestIDHeader))
			require.Equal(t, identifiers["correlation_id"], resp.Header.Get(middleware.CorrelationIDHeader))

			traceID, err := trace.TraceIDFromHex(identifiers["trace_id"])
			require.NoError(t, err)
			require.True(t, traceID.IsValid())
			if tt.traceparent != "" {
				require.Equal(t, incomingTraceID, traceID.String())
			}

			var entry map[string]any
			require.NoError(t, json.NewDecoder(&logs).Decode(&entry))
			for _, key := range []string{"request_id", "correlation_id", "trace_id"} {
				require.Equal(t, identifiers[key], entry[key], key)
			}
		})
	}
}
