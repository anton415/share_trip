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

func TestServer_PublishTrip(t *testing.T) {
	t.Run("validation error - идентификаторы должны быть UUID", func(t *testing.T) {
		tests := []struct {
			name    string
			payload api.PublishTripRequest
		}{
			{
				name: "invalid trip id",
				payload: api.PublishTripRequest{
					TripID:   "not-a-uuid",
					ClientID: uuid.NewString(),
				},
			},
			{
				name: "invalid client id",
				payload: api.PublishTripRequest{
					TripID:   uuid.NewString(),
					ClientID: "",
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				resp := sendPublishTrip(t, tt.payload)
				defer closeResponseBody(t, resp.Body)

				require.Equal(t, http.StatusBadRequest, resp.StatusCode)
				requireErrorResponse(t, resp, "VALIDATION_ERROR", "invalid request body")
			})
		}
	})

	t.Run("success - перевод поездки из draft в published", func(t *testing.T) {
		created := createDraftTrip(t)

		publishResp := sendPublishTrip(t, api.PublishTripRequest{
			TripID:   created.ID,
			ClientID: created.DriverID,
		})
		defer closeResponseBody(t, publishResp.Body)

		require.Equal(t, http.StatusOK, publishResp.StatusCode)

		publishRespBody, err := io.ReadAll(publishResp.Body)
		require.NoError(t, err)

		var published api.PublishTripResponse
		require.NoError(t, json.Unmarshal(publishRespBody, &published))
		require.Equal(t, api.PublishTripResponse{
			TripID: created.ID,
		}, published)

		getReq, err := http.NewRequest(http.MethodGet, "/trip/"+created.ID, nil)
		require.NoError(t, err)

		getResp, err := testApp.Test(getReq, -1)
		require.NoError(t, err)
		defer closeResponseBody(t, getResp.Body)

		require.Equal(t, http.StatusOK, getResp.StatusCode)

		getRespBody, err := io.ReadAll(getResp.Body)
		require.NoError(t, err)

		var got api.CreateTripResponse
		require.NoError(t, json.Unmarshal(getRespBody, &got))

		require.False(t, got.UpdatedAt.IsZero())
		require.False(t, got.UpdatedAt.Before(created.UpdatedAt))

		expected := created
		expected.Status = domain.TripStatusPublished
		expected.UpdatedAt = got.UpdatedAt
		require.Equal(t, expected, got)
	})

	t.Run("forbidden - client не является водителем поездки", func(t *testing.T) {
		created := createDraftTrip(t)

		resp := sendPublishTrip(t, api.PublishTripRequest{
			TripID:   created.ID,
			ClientID: uuid.NewString(),
		})
		defer closeResponseBody(t, resp.Body)

		require.Equal(t, http.StatusForbidden, resp.StatusCode)
		requireErrorResponse(t, resp, "FORBIDDEN", "forbidden")
	})

	t.Run("not found - поездка не существует", func(t *testing.T) {
		resp := sendPublishTrip(t, api.PublishTripRequest{
			TripID:   uuid.NewString(),
			ClientID: uuid.NewString(),
		})
		defer closeResponseBody(t, resp.Body)

		require.Equal(t, http.StatusNotFound, resp.StatusCode)
		requireErrorResponse(t, resp, "NOT_FOUND", "trip not found")
	})

	t.Run("conflict - поездка не в статусе draft", func(t *testing.T) {
		created := createDraftTrip(t)

		_, err := testPool.Exec(
			testCtx,
			`UPDATE trips SET status = 'cancelled' WHERE id = $1`,
			created.ID,
		)
		require.NoError(t, err)

		resp := sendPublishTrip(t, api.PublishTripRequest{
			TripID:   created.ID,
			ClientID: created.DriverID,
		})
		defer closeResponseBody(t, resp.Body)

		require.Equal(t, http.StatusConflict, resp.StatusCode)
		requireErrorResponse(t, resp, "CONFLICT", "trip is not in draft status")
	})

	t.Run("no content - поездка уже published", func(t *testing.T) {
		created := createDraftTrip(t)
		payload := api.PublishTripRequest{
			TripID:   created.ID,
			ClientID: created.DriverID,
		}

		firstResp := sendPublishTrip(t, payload)
		require.Equal(t, http.StatusOK, firstResp.StatusCode)
		require.NoError(t, firstResp.Body.Close())

		secondResp := sendPublishTrip(t, payload)
		defer closeResponseBody(t, secondResp.Body)

		require.Equal(t, http.StatusNoContent, secondResp.StatusCode)
		body, err := io.ReadAll(secondResp.Body)
		require.NoError(t, err)
		require.Empty(t, body)

		var eventCount int
		err = testPool.QueryRow(
			testCtx,
			`SELECT COUNT(*) FROM outbox_event WHERE aggregate_id = $1 AND event_name = 'trip_published'`,
			created.ID,
		).Scan(&eventCount)
		require.NoError(t, err)
		require.Equal(t, 1, eventCount)
	})
}

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

func sendPublishTrip(t *testing.T, payload api.PublishTripRequest) *http.Response {
	t.Helper()

	body, err := json.Marshal(payload)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, "/trip/publish", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := testApp.Test(req, -1)
	require.NoError(t, err)

	return resp
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
