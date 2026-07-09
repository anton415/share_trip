package api_test

import (
	"bytes"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"io"
	"job4j.ru/share-trip/internal/api"
	"net/http"
	"testing"
	"time"
)

func TestServer_CreateTrip(t *testing.T) {
	t.Run("success - создание поездки", func(t *testing.T) {
		payload := api.CreateTripRequest{
			DriverID:       uuid.NewString(),
			FromPoint:      "Moscow",
			ToPoint:        "Saint Petersburg",
			DepartureTime:  time.Now().Add(24 * time.Hour),
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
		require.Equal(t, payload.DriverID, got.DriverID)
		require.Equal(t, payload.FromPoint, got.FromPoint)
		require.Equal(t, payload.ToPoint, got.ToPoint)
		require.Equal(t, payload.AvailableSeats, got.AvailableSeats)
		require.Equal(t, "draft", string(got.Status))
		require.False(t, got.CreatedAt.IsZero())
		require.False(t, got.UpdatedAt.IsZero())
	})
}
