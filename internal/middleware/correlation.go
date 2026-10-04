package middleware

import (
	"context"
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"

	"job4j.ru/share-trip/internal/observability/logctx"
)

const (
	RequestIDHeader     = "X-Request-Id"
	CorrelationIDHeader = "X-Correlation-ID"
	LoggerLocalKey      = "logger"
)

func Correlation(baseLogger *slog.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		requestID := c.Get(RequestIDHeader)
		if requestID == "" {
			requestID = uuid.NewString()
		}
		correlationID := c.Get(CorrelationIDHeader)
		if correlationID == "" {
			correlationID = requestID
		}

		c.Set(RequestIDHeader, requestID)
		c.Set(CorrelationIDHeader, correlationID)

		requestLogger := baseLogger.With(
			slog.String("request_id", requestID),
			slog.String("correlation_id", correlationID),
			slog.String("method", c.Method()),
			slog.String("path", c.Path()),
		)

		ctx := c.UserContext()
		if ctx == nil {
			ctx = context.Background()
		}
		if spanContext := trace.SpanContextFromContext(ctx); spanContext.IsValid() {
			requestLogger = requestLogger.With(slog.String("trace_id", spanContext.TraceID().String()))
		}

		ctx = logctx.WithRequestID(ctx, requestID)
		ctx = logctx.WithCorrelationID(ctx, correlationID)
		ctx = logctx.WithLogger(ctx, requestLogger)

		c.SetUserContext(ctx)
		c.Locals(LoggerLocalKey, requestLogger)

		return c.Next()
	}
}
