package middleware

import (
	"context"
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"job4j.ru/share-trip/internal/observability/logctx"
)

const (
	RequestIDHeader = "X-Request-Id"
	LoggerLocalKey  = "logger"
)

func Correlation(baseLogger *slog.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		requestID := c.Get(RequestIDHeader)
		if requestID == "" {
			requestID = uuid.NewString()
		}

		c.Set(RequestIDHeader, requestID)

		requestLogger := baseLogger.With(
			slog.String("request_id", requestID),
			slog.String("method", c.Method()),
			slog.String("path", c.Path()),
		)

		ctx := c.UserContext()
		if ctx == nil {
			ctx = context.Background()
		}

		ctx = logctx.WithRequestID(ctx, requestID)
		ctx = logctx.WithLogger(ctx, requestLogger)

		c.SetUserContext(ctx)
		c.Locals(LoggerLocalKey, requestLogger)

		return c.Next()
	}
}
