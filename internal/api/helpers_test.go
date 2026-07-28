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

func createDraftTrip(t *testing.T) api.CreateTripResponse {
	t.Helper()

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

	req, err := http.NewRequest(http.MethodPost, "/trip/create", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := testApp.Test(req, -1)
	require.NoError(t, err)
	defer closeResponseBody(t, resp.Body)

	require.Equal(t, http.StatusCreated, resp.StatusCode)

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var created api.CreateTripResponse
	require.NoError(t, json.Unmarshal(respBody, &created))

	require.NotEmpty(t, created.ID)
	require.False(t, created.CreatedAt.IsZero())
	require.False(t, created.UpdatedAt.IsZero())
	require.WithinDuration(t, payload.DepartureTime, created.DepartureTime, time.Microsecond)
	require.Equal(t, api.CreateTripResponse{
		ID:             created.ID,
		DriverID:       payload.DriverID,
		FromPoint:      payload.FromPoint,
		ToPoint:        payload.ToPoint,
		DepartureTime:  created.DepartureTime,
		AvailableSeats: payload.AvailableSeats,
		Status:         domain.TripStatusDraft,
		CreatedAt:      created.CreatedAt,
		UpdatedAt:      created.UpdatedAt,
	}, created)

	return created
}

func requireErrorResponse(t *testing.T, resp *http.Response, code, message string) {
	t.Helper()

	type errorResponse struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}

	var got errorResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&got))
	require.Equal(t, errorResponse{
		Code:    code,
		Message: message,
	}, got)
}

func closeResponseBody(t *testing.T, body io.Closer) {
	t.Helper()

	if err := body.Close(); err != nil {
		t.Errorf("close response body: %v", err)
	}
}
