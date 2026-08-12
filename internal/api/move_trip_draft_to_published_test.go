package api_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"job4j.ru/share-trip/internal/api"
	"job4j.ru/share-trip/internal/domain"
)

func TestServer_MoveTripDraftToPublished(t *testing.T) {
	t.Run("validation error - идентификаторы должны быть UUID", func(t *testing.T) {
		tests := []struct {
			name    string
			payload api.MoveTripDraftToPublishedRequest
		}{
			{
				name: "invalid trip id",
				payload: api.MoveTripDraftToPublishedRequest{
					TripID:   "not-a-uuid",
					ClientID: uuid.NewString(),
				},
			},
			{
				name: "invalid client id",
				payload: api.MoveTripDraftToPublishedRequest{
					TripID:   uuid.NewString(),
					ClientID: "",
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				resp := sendMoveTripDraftToPublished(t, tt.payload)
				defer closeResponseBody(t, resp.Body)

				require.Equal(t, http.StatusBadRequest, resp.StatusCode)
				requireErrorResponse(t, resp, "VALIDATION_ERROR", "invalid request body")
			})
		}
	})

	t.Run("success - перевод поездки из draft в published", func(t *testing.T) {
		created := createDraftTrip(t)

		publishResp := sendMoveTripDraftToPublished(t, api.MoveTripDraftToPublishedRequest{
			TripID:   created.ID,
			ClientID: created.DriverID,
		})
		defer closeResponseBody(t, publishResp.Body)

		require.Equal(t, http.StatusOK, publishResp.StatusCode)

		publishRespBody, err := io.ReadAll(publishResp.Body)
		require.NoError(t, err)

		var published api.MoveTripDraftToPublishedResponse
		require.NoError(t, json.Unmarshal(publishRespBody, &published))
		require.Equal(t, api.MoveTripDraftToPublishedResponse{
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

		var got api.GetTripByIDResponse
		require.NoError(t, json.Unmarshal(getRespBody, &got))

		require.False(t, got.UpdatedAt.IsZero())
		require.False(t, got.UpdatedAt.Before(created.UpdatedAt))

		expected := api.GetTripByIDResponse(created)
		expected.Status = domain.TripStatusPublished
		expected.UpdatedAt = got.UpdatedAt
		require.Equal(t, expected, got)
	})

	t.Run("forbidden - client не является водителем поездки", func(t *testing.T) {
		created := createDraftTrip(t)

		resp := sendMoveTripDraftToPublished(t, api.MoveTripDraftToPublishedRequest{
			TripID:   created.ID,
			ClientID: uuid.NewString(),
		})
		defer closeResponseBody(t, resp.Body)

		require.Equal(t, http.StatusForbidden, resp.StatusCode)
		requireErrorResponse(t, resp, "FORBIDDEN", "forbidden")
	})

	t.Run("not found - поездка не существует", func(t *testing.T) {
		resp := sendMoveTripDraftToPublished(t, api.MoveTripDraftToPublishedRequest{
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

		resp := sendMoveTripDraftToPublished(t, api.MoveTripDraftToPublishedRequest{
			TripID:   created.ID,
			ClientID: created.DriverID,
		})
		defer closeResponseBody(t, resp.Body)

		require.Equal(t, http.StatusConflict, resp.StatusCode)
		requireErrorResponse(t, resp, "CONFLICT", "trip is not in draft status")
	})

	t.Run("no content - поездка уже published", func(t *testing.T) {
		created := createDraftTrip(t)
		payload := api.MoveTripDraftToPublishedRequest{
			TripID:   created.ID,
			ClientID: created.DriverID,
		}

		firstResp := sendMoveTripDraftToPublished(t, payload)
		require.Equal(t, http.StatusOK, firstResp.StatusCode)
		require.NoError(t, firstResp.Body.Close())

		secondResp := sendMoveTripDraftToPublished(t, payload)
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

func sendMoveTripDraftToPublished(
	t *testing.T,
	payload api.MoveTripDraftToPublishedRequest,
) *http.Response {
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
