package api_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"job4j.ru/share-trip/internal/api"
	"job4j.ru/share-trip/internal/domain"
)

func TestServer_MoveTripDraftToPublished(t *testing.T) {
	requireIntegration(t)
	t.Parallel()

	t.Run("validation error - обязательные идентификаторы отсутствуют", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name            string
			payload         api.MoveTripDraftToPublishedRequest
			expectedMessage string
		}{
			{
				name: "trip id is empty",
				payload: api.MoveTripDraftToPublishedRequest{
					TripID:   "",
					ClientID: uuid.NewString(),
				},
				expectedMessage: "tripId is required",
			},
			{
				name: "client id contains only spaces",
				payload: api.MoveTripDraftToPublishedRequest{
					TripID:   uuid.NewString(),
					ClientID: "   ",
				},
				expectedMessage: "clientId is required",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				fixture := newTestFixture()
				resp := sendMoveTripDraftToPublished(t, fixture.app, tt.payload)
				defer closeResponseBody(t, resp.Body)

				require.Equal(t, http.StatusBadRequest, resp.StatusCode)
				requireErrorResponse(
					t,
					resp,
					"VALIDATION_ERROR",
					tt.expectedMessage,
				)
			})
		}
	})

	t.Run("validation error - идентификаторы должны быть UUID", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name            string
			payload         api.MoveTripDraftToPublishedRequest
			expectedMessage string
		}{
			{
				name: "invalid trip id",
				payload: api.MoveTripDraftToPublishedRequest{
					TripID:   "not-a-uuid",
					ClientID: uuid.NewString(),
				},
				expectedMessage: "tripId must be a valid UUID",
			},
			{
				name: "invalid client id",
				payload: api.MoveTripDraftToPublishedRequest{
					TripID:   uuid.NewString(),
					ClientID: "not-a-uuid",
				},
				expectedMessage: "clientId must be a valid UUID",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				fixture := newTestFixture()
				resp := sendMoveTripDraftToPublished(t, fixture.app, tt.payload)
				defer closeResponseBody(t, resp.Body)

				require.Equal(t, http.StatusBadRequest, resp.StatusCode)
				requireErrorResponse(
					t,
					resp,
					"VALIDATION_ERROR",
					tt.expectedMessage,
				)
			})
		}
	})

	t.Run("validation error - тело запроса должно быть валидным JSON", func(t *testing.T) {
		t.Parallel()

		fixture := newTestFixture()
		req, err := http.NewRequest(
			http.MethodPost,
			"/trip/publish",
			strings.NewReader("{"),
		)
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := fixture.app.Test(req, -1)
		require.NoError(t, err)
		defer closeResponseBody(t, resp.Body)

		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		requireErrorResponse(t, resp, "VALIDATION_ERROR", "invalid request body")
	})

	t.Run("success - перевод поездки из draft в published", func(t *testing.T) {
		t.Parallel()

		fixture := newTestFixture()
		created := createDraftTrip(t, fixture.app)

		publishResp := sendMoveTripDraftToPublished(t, fixture.app, api.MoveTripDraftToPublishedRequest{
			TripID:   created.ID.String(),
			ClientID: created.DriverID.String(),
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

		getReq, err := http.NewRequest(http.MethodGet, "/trip/"+created.ID.String(), nil)
		require.NoError(t, err)

		getResp, err := fixture.app.Test(getReq, -1)
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
		t.Parallel()

		fixture := newTestFixture()
		created := createDraftTrip(t, fixture.app)

		resp := sendMoveTripDraftToPublished(t, fixture.app, api.MoveTripDraftToPublishedRequest{
			TripID:   created.ID.String(),
			ClientID: uuid.NewString(),
		})
		defer closeResponseBody(t, resp.Body)

		require.Equal(t, http.StatusForbidden, resp.StatusCode)
		requireErrorResponse(t, resp, "FORBIDDEN", "forbidden")
	})

	t.Run("not found - поездка не существует", func(t *testing.T) {
		t.Parallel()

		fixture := newTestFixture()
		resp := sendMoveTripDraftToPublished(t, fixture.app, api.MoveTripDraftToPublishedRequest{
			TripID:   uuid.NewString(),
			ClientID: uuid.NewString(),
		})
		defer closeResponseBody(t, resp.Body)

		require.Equal(t, http.StatusNotFound, resp.StatusCode)
		requireErrorResponse(t, resp, "NOT_FOUND", "trip not found")
	})

	t.Run("conflict - поездка не в статусе draft", func(t *testing.T) {
		t.Parallel()

		fixture := newTestFixture()
		created := createDraftTrip(t, fixture.app)

		_, err := testPool.Exec(
			testCtx,
			`UPDATE trips SET status = 'cancelled' WHERE id = $1`,
			created.ID,
		)
		require.NoError(t, err)

		resp := sendMoveTripDraftToPublished(t, fixture.app, api.MoveTripDraftToPublishedRequest{
			TripID:   created.ID.String(),
			ClientID: created.DriverID.String(),
		})
		defer closeResponseBody(t, resp.Body)

		require.Equal(t, http.StatusConflict, resp.StatusCode)
		requireErrorResponse(t, resp, "CONFLICT", "trip is not in draft status")
	})

	t.Run("no content - поездка уже published", func(t *testing.T) {
		t.Parallel()

		fixture := newTestFixture()
		created := createDraftTrip(t, fixture.app)
		payload := api.MoveTripDraftToPublishedRequest{
			TripID:   created.ID.String(),
			ClientID: created.DriverID.String(),
		}

		firstResp := sendMoveTripDraftToPublished(t, fixture.app, payload)
		require.Equal(t, http.StatusOK, firstResp.StatusCode)
		require.NoError(t, firstResp.Body.Close())

		secondResp := sendMoveTripDraftToPublished(t, fixture.app, payload)
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
	app *fiber.App,
	payload api.MoveTripDraftToPublishedRequest,
) *http.Response {
	t.Helper()

	body, err := json.Marshal(payload)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, "/trip/publish", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	require.NoError(t, err)

	return resp
}
