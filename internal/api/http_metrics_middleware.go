package api

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"job4j.ru/share-trip/internal/observability/metrics"
)

func NewHTTPMetricsMiddleware(m *metrics.Metrics) fiber.Handler {
	return func(c *fiber.Ctx) error {
		middlewareRoute := c.Route()
		started := time.Now()

		err := c.Next()

		path := c.Route().Path
		if path == "" || c.Route() == middlewareRoute {
			path = "unmatched"
		}

		statusCode := c.Response().StatusCode()
		var fiberErr *fiber.Error

		switch {
		case errors.As(err, &fiberErr):
			statusCode = fiberErr.Code
		case err != nil:
			statusCode = fiber.StatusInternalServerError
		}

		method := strings.Clone(c.Method())
		status := strconv.Itoa(statusCode)

		m.HTTPRequestTotal.
			WithLabelValues(method, path, status).
			Inc()

		m.HTTPRequestDuration.
			WithLabelValues(method, path, status).
			Observe(time.Since(started).Seconds())

		return err
	}
}
