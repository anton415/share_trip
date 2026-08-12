package api

import (
	"github.com/gofiber/adaptor/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func (s *Server) metricsHandler() fiber.Handler {
	return adaptor.HTTPHandler(
		promhttp.HandlerFor(s.registry, promhttp.HandlerOpts{}),
	)
}
