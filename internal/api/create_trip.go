package api

import (
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"job4j.ru/share-trip/internal/domain"
	"job4j.ru/share-trip/internal/observability/logctx"
	"job4j.ru/share-trip/internal/service"
)

type CreateTripRequest struct {
	FromPoint      string    `json:"fromPoint"`
	ToPoint        string    `json:"toPoint"`
	DepartureTime  time.Time `json:"departureTime"`
	AvailableSeats int       `json:"availableSeats"`
}

type CreateTripResponse struct {
	ID             uuid.UUID         `json:"id"`
	DriverID       uuid.UUID         `json:"driverId"`
	FromPoint      string            `json:"fromPoint"`
	ToPoint        string            `json:"toPoint"`
	DepartureTime  time.Time         `json:"departureTime"`
	AvailableSeats int               `json:"availableSeats"`
	Status         domain.TripStatus `json:"status"`
	CreatedAt      time.Time         `json:"createdAt"`
	UpdatedAt      time.Time         `json:"updatedAt"`
}

func (s *Server) createTrip(c *fiber.Ctx) error {
	ctx := c.UserContext()
	driverID, err := clientIDFromClaims(c)
	if err != nil {
		return err
	}

	logger := logctx.Logger(ctx).With(
		slog.String("server", "TripServer"),
		slog.String("handler", "CreateTrip"),
		slog.String("client_id", driverID.String()),
	)

	var request CreateTripRequest

	if err := c.BodyParser(&request); err != nil {
		logger.Warn(
			"create trip failed: invalid json body",
			slog.Any("error", err),
		)
		return c.Status(fiber.StatusBadRequest).JSON(errorResponse{
			Code:    "VALIDATION_ERROR",
			Message: "invalid request body",
		})
	}

	ctx = logctx.WithLogger(ctx, logger)

	logger.Info("create trip request accepted")

	trip, err := s.trips.CreateTrip(ctx, service.CreateTripCommand{
		DriverID:      driverID,
		FromPoint:     request.FromPoint,
		ToPoint:       request.ToPoint,
		DepartureTime: request.DepartureTime,
		Seats:         request.AvailableSeats,
	})
	if err != nil {
		logger.Error(
			"create trip failed",
			slog.Any("error", err),
		)
		return writeFiberServiceError(c, err)
	}

	logger.Info(
		"create trip completed",
		slog.String("trip_id", trip.ID.String()),
	)

	return c.Status(fiber.StatusCreated).JSON(newCreateTripResponse(trip))
}
