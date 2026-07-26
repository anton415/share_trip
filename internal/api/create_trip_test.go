package api_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"job4j.ru/share-trip/internal/api"
	"job4j.ru/share-trip/internal/domain"
)

func TestServer_CreateTrip(t *testing.T) {
	t.Run("success - создание поездки", func(t *testing.T) {
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

		body, err := json.Marshal(payload)
		require.NoError(t, err)

		req, err := http.NewRequest(
			http.MethodPost,
			"/trip/create",
			bytes.NewReader(body),
		)
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := testApp.Test(req, -1)
		require.NoError(t, err)
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
			DriverID:       payload.DriverID,
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
