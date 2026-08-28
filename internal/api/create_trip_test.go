package api_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"job4j.ru/share-trip/internal/api"
	"job4j.ru/share-trip/internal/domain"
)

func TestServer_CreateTrip(t *testing.T) {
	requireIntegration(t)
	t.Parallel()

	t.Run("validation error - идентификатор водителя обязателен", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name     string
			driverID string
		}{
			{name: "only spaces", driverID: "   "},
			{name: "nil UUID", driverID: uuid.Nil.String()},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				fixture := newTestFixture()
				payload := api.CreateTripRequest{
					DriverID:       tt.driverID,
					FromPoint:      "Moscow",
					ToPoint:        "Saint Petersburg",
					DepartureTime:  time.Now().UTC().Add(24 * time.Hour),
					AvailableSeats: 3,
				}

				resp := sendCreateTrip(t, fixture.app, payload)
				defer closeResponseBody(t, resp.Body)

				require.Equal(t, http.StatusBadRequest, resp.StatusCode)
				requireErrorResponse(t, resp, "VALIDATION_ERROR", "driverId is required")
			})
		}
	})

	t.Run("validation error - идентификатор водителя должен быть UUID", func(t *testing.T) {
		t.Parallel()

		fixture := newTestFixture()
		payload := api.CreateTripRequest{
			DriverID:       "not-a-uuid",
			FromPoint:      "Moscow",
			ToPoint:        "Saint Petersburg",
			DepartureTime:  time.Now().UTC().Add(24 * time.Hour),
			AvailableSeats: 3,
		}

		resp := sendCreateTrip(t, fixture.app, payload)
		defer closeResponseBody(t, resp.Body)

		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		requireErrorResponse(t, resp, "VALIDATION_ERROR", "driverId must be a valid UUID")
	})

	t.Run("success - создание поездки", func(t *testing.T) {
		t.Parallel()

		fixture := newTestFixture()
		departureTime := time.Now().
			UTC().
			Add(24 * time.Hour).
			Truncate(time.Microsecond)

		payload := api.CreateTripRequest{
			DriverID:       uuid.NewString(),
			FromPoint:      "Moscow",
			ToPoint:        "Saint Petersburg",
			DepartureTime:  departureTime,
			AvailableSeats: 3,
		}

		resp := sendCreateTrip(t, fixture.app, payload)
		defer func() {
			if err := resp.Body.Close(); err != nil {
				t.Errorf("close response body: %v", err)
			}
		}()

		require.Equal(t, http.StatusCreated, resp.StatusCode)

		respBody, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		var got api.CreateTripResponse
		err = json.Unmarshal(respBody, &got)
		require.NoError(t, err)

		require.NotEmpty(t, got.ID)
		require.False(t, got.CreatedAt.IsZero())
		require.False(t, got.UpdatedAt.IsZero())
		require.WithinDuration(t, payload.DepartureTime, got.DepartureTime, time.Microsecond)
		require.Equal(t, api.CreateTripResponse{
			ID:             got.ID,
			DriverID:       uuid.MustParse(payload.DriverID),
			FromPoint:      payload.FromPoint,
			ToPoint:        payload.ToPoint,
			DepartureTime:  got.DepartureTime,
			AvailableSeats: payload.AvailableSeats,
			Status:         domain.TripStatusDraft,
			CreatedAt:      got.CreatedAt,
			UpdatedAt:      got.UpdatedAt,
		}, got)
	})
}

func sendCreateTrip(
	t *testing.T,
	app *fiber.App,
	payload api.CreateTripRequest,
) *http.Response {
	t.Helper()

	body, err := json.Marshal(payload)
	require.NoError(t, err)

	req, err := http.NewRequest(
		http.MethodPost,
		"/trip/create",
		bytes.NewReader(body),
	)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	require.NoError(t, err)

	return resp
}
