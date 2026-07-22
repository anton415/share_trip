package api

import (
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"job4j.ru/share-trip/internal/domain"
	"job4j.ru/share-trip/internal/observability/logctx"
	"job4j.ru/share-trip/internal/service"
)

type PublishTripRequest struct {
	TripID   string `json:"tripId"`
	ClientID string `json:"clientId"`
}

type PublishTripResponse struct {
	TripID string `json:"tripId"`
}

type CreateTripRequest struct {
	DriverID       string    `json:"driverId"`
	FromPoint      string    `json:"fromPoint"`
	ToPoint        string    `json:"toPoint"`
	DepartureTime  time.Time `json:"departureTime"`
	AvailableSeats int       `json:"availableSeats"`
}

type CreateTripResponse struct {
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

type errorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func newTripResponse(trip domain.Trip) CreateTripResponse {
	return CreateTripResponse{
		ID:             trip.ID,
		DriverID:       trip.DriverID,
		FromPoint:      trip.FromPoint,
		ToPoint:        trip.ToPoint,
		DepartureTime:  trip.DepartureTime,
		AvailableSeats: trip.Seats,
		Status:         trip.Status,
		CreatedAt:      trip.CreatedAt,
		UpdatedAt:      trip.UpdatedAt,
	}
}

func (s *Server) createTrip(c *fiber.Ctx) error {
	ctx := c.UserContext()

	logger := logctx.Logger(ctx).With(
		slog.String("server", "TripServer"),
		slog.String("handler", "CreateTrip"),
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

	logger = logger.With(
		slog.String("client_id", request.DriverID),
	)

	ctx = logctx.WithLogger(ctx, logger)

	logger.Info("create trip request accepted")

	trip, err := s.trips.CreateTrip(ctx, service.CreateTripCommand{
		DriverID:      request.DriverID,
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
		slog.String("trip_id", trip.ID),
	)

	return c.Status(fiber.StatusCreated).JSON(newTripResponse(trip))
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

func (s *Server) publishTrip(c *fiber.Ctx) error {
	var request PublishTripRequest

	if err := c.BodyParser(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errorResponse{
			Code:    "VALIDATION_ERROR",
			Message: "invalid request body",
		})
	}

	tripID, err := s.trips.PublishTrip(c.UserContext(), service.PublishTripCommand{
		TripID:   request.TripID,
		ClientID: request.ClientID,
	})
	if errors.Is(err, domain.ErrTripAlreadyPublished) {
		return c.SendStatus(fiber.StatusNoContent)
	}
	if err != nil {
		return writeFiberServiceError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(PublishTripResponse{
		TripID: tripID,
	})
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

	return c.Status(fiber.StatusOK).JSON(newTripResponse(trip))
}
