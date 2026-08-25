package api

import "job4j.ru/share-trip/internal/domain"

func newGetTripByIDResponse(trip domain.Trip) GetTripByIDResponse {
	return GetTripByIDResponse{
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
