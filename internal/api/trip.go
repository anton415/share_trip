package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"job4j.ru/share-trip/internal/domain"
	"job4j.ru/share-trip/internal/service"
)

type publishTripRequest struct {
	TripID   string `json:"tripId"`
	ClientID string `json:"clientId"`
}

type publishTripResponse struct {
	TripID string `json:"tripId"`
}

type createTripRequest struct {
	DriverID       string    `json:"driverId"`
	FromPoint      string    `json:"fromPoint"`
	ToPoint        string    `json:"toPoint"`
	DepartureTime  time.Time `json:"departureTime"`
	AvailableSeats int       `json:"availableSeats"`
}

type tripResponse struct {
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

func createTripHandler(trips TripService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var request createTripRequest
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()

		if err := decoder.Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
			return
		}

		trip, err := trips.CreateTrip(r.Context(), service.CreateTripCommand{
			DriverID:      request.DriverID,
			FromPoint:     request.FromPoint,
			ToPoint:       request.ToPoint,
			DepartureTime: request.DepartureTime,
			Seats:         request.AvailableSeats,
		})
		if err != nil {
			handleServiceError(w, err)
			return
		}

		writeJSON(w, http.StatusCreated, newTripResponse(trip))
	}
}

func getTripByIDHandler(trips TripService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		tripID := strings.TrimPrefix(r.URL.Path, "/trip/")
		if tripID == "" || strings.Contains(tripID, "/") {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "trip not found")
			return
		}

		trip, err := trips.GetTripByID(r.Context(), tripID)
		if err != nil {
			handleServiceError(w, err)
			return
		}

		writeJSON(w, http.StatusOK, newTripResponse(trip))
	}
}

func newTripResponse(trip domain.Trip) tripResponse {
	return tripResponse{
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

func handleServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrValidation):
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
	case errors.Is(err, service.ErrNotFound):
		writeError(w, http.StatusNotFound, "NOT_FOUND", "trip not found")
	default:
		slog.Error("trip request failed", "error", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
	}
}

func writeJSON(w http.ResponseWriter, statusCode int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, statusCode int, code string, message string) {
	writeJSON(w, statusCode, errorResponse{
		Code:    code,
		Message: message,
	})
}

func publishTripHandler(trips TripService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var request publishTripRequest
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()

		if err := decoder.Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
			return
		}

		tripID, err := trips.PublishTrip(r.Context(), service.PublishTripCommand{
			TripID:   request.TripID,
			ClientID: request.ClientID,
		})
		if err != nil {
			handleServiceError(w, err)
			return
		}

		writeJSON(w, http.StatusOK, publishTripResponse{TripID: tripID})
	}
}
