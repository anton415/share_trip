package api

import (
	"errors"
	"log/slog"

	"github.com/gofiber/fiber/v2"

	"job4j.ru/share-trip/internal/domain"
	"job4j.ru/share-trip/internal/service"
)

type errorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeFiberServiceError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, service.ErrValidation):
		return c.Status(fiber.StatusBadRequest).JSON(errorResponse{
			Code:    "VALIDATION_ERROR",
			Message: err.Error(),
		})
	case errors.Is(err, domain.ErrTripNotFound):
		return c.Status(fiber.StatusNotFound).JSON(errorResponse{
			Code:    "NOT_FOUND",
			Message: "trip not found",
		})
	case errors.Is(err, domain.ErrForbidden):
		return c.Status(fiber.StatusForbidden).JSON(errorResponse{
			Code:    "FORBIDDEN",
			Message: "forbidden",
		})
	case errors.Is(err, domain.ErrConflict):
		return c.Status(fiber.StatusConflict).JSON(errorResponse{
			Code:    "CONFLICT",
			Message: "trip is not in draft status",
		})
	default:
		slog.Error("trip request failed", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(errorResponse{
			Code:    "INTERNAL_ERROR",
			Message: "internal server error",
		})
	}
}
