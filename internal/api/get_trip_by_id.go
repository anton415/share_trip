package api

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"job4j.ru/share-trip/internal/domain"
)

type GetTripByIDResponse struct {
	ID             string            `json:"id"`
	DriverID       string            `json:"driverId"`
	FromPoint      string            `json:"fromPoint"`
	ToPoint        string            `json:"toPoint"`
	DepartureTime  time.Time         `json:"departureTime"`
	AvailableSeats int               `json:"availableSeats"`
	Status         domain.TripStatus `json:"status"`
	CreatedAt      time.Time         `json:"createdAt"`
	UpdatedAt      time.Time         `json:"updatedAt"`
}

func (s *Server) getTripByID(c *fiber.Ctx) error {
	tripID := c.Params("id")
	if strings.TrimSpace(tripID) == "" {
		return c.Status(fiber.StatusNotFound).JSON(errorResponse{
			Code:    "NOT_FOUND",
			Message: "trip not found",
		})
	}

	trip, err := s.trips.GetTripByID(c.UserContext(), tripID)
	if err != nil {
		return writeFiberServiceError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(newGetTripByIDResponse(trip))
}
