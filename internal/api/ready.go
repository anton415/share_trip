package api

import (
	"context"
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v2"
)

func (s *Server) ready(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.UserContext(), 3*time.Second)
	defer cancel()

	if err := s.db.Ping(ctx); err != nil {
		slog.Error("database readiness check failed", "error", err)
		return c.Status(fiber.StatusServiceUnavailable).JSON(errorResponse{
			Code:    "SERVICE_UNAVAILABLE",
			Message: "database unavailable",
		})
	}

	return c.SendStatus(fiber.StatusOK)
}
