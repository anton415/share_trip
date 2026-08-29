package api_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"job4j.ru/share-trip/internal/api"
)

func TestServer_GetTripByID(t *testing.T) {
	requireIntegration(t)
	t.Parallel()

	t.Run("validation error - идентификатор должен быть UUID", func(t *testing.T) {
		t.Parallel()

		fixture := newTestFixture()
		req, err := http.NewRequest(http.MethodGet, "/trip/not-a-uuid", nil)
		require.NoError(t, err)

		resp, err := fixture.app.Test(req, -1)
		require.NoError(t, err)
		defer closeResponseBody(t, resp.Body)

		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		requireErrorResponse(t, resp, "VALIDATION_ERROR", "tripId must be a valid UUID")
	})

	t.Run("success - получение поездки по идентификатору", func(t *testing.T) {
		t.Parallel()

		fixture := newTestFixture()
		created := createDraftTrip(t, fixture)

		req, err := http.NewRequest(http.MethodGet, "/trip/"+created.ID.String(), nil)
		require.NoError(t, err)

		resp, err := fixture.app.Test(req, -1)
		require.NoError(t, err)
		defer closeResponseBody(t, resp.Body)

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var got api.GetTripByIDResponse
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&got))
		require.Equal(t, api.GetTripByIDResponse(created), got)
	})

	t.Run("not found - поездка не существует", func(t *testing.T) {
		t.Parallel()

		fixture := newTestFixture()
		req, err := http.NewRequest(http.MethodGet, "/trip/"+uuid.NewString(), nil)
		require.NoError(t, err)

		resp, err := fixture.app.Test(req, -1)
		require.NoError(t, err)
		defer closeResponseBody(t, resp.Body)

		require.Equal(t, http.StatusNotFound, resp.StatusCode)
		requireErrorResponse(t, resp, "NOT_FOUND", "trip not found")
	})
}
