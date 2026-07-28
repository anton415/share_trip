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
	t.Run("success - получение поездки по идентификатору", func(t *testing.T) {
		created := createDraftTrip(t)

		req, err := http.NewRequest(http.MethodGet, "/trip/"+created.ID, nil)
		require.NoError(t, err)

		resp, err := testApp.Test(req, -1)
		require.NoError(t, err)
		defer closeResponseBody(t, resp.Body)

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var got api.CreateTripResponse
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&got))
		require.Equal(t, created, got)
	})

	t.Run("not found - поездка не существует", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "/trip/"+uuid.NewString(), nil)
		require.NoError(t, err)

		resp, err := testApp.Test(req, -1)
		require.NoError(t, err)
		defer closeResponseBody(t, resp.Body)

		require.Equal(t, http.StatusNotFound, resp.StatusCode)
		requireErrorResponse(t, resp, "NOT_FOUND", "trip not found")
	})
}
