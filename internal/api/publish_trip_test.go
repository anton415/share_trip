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
)

func TestServer_PublishTrip(t *testing.T) {
	t.Run("success - перевод поездки из draft в published", func(t *testing.T) {
		createPayload := api.CreateTripRequest{
			DriverID:       uuid.NewString(),
			FromPoint:      "Moscow",
			ToPoint:        "Saint Petersburg",
			DepartureTime:  time.Now().Add(24 * time.Hour),
			AvailableSeats: 3,
		}

		createBody, err := json.Marshal(createPayload)
		require.NoError(t, err)

		createReq, err := http.NewRequest(
			http.MethodPost,
			"/trip/create",
			bytes.NewReader(createBody),
		)
		require.NoError(t, err)
		createReq.Header.Set("Content-Type", "application/json")

		createResp, err := testApp.Test(createReq, -1)
		require.NoError(t, err)
		defer createResp.Body.Close()

		require.Equal(t, http.StatusCreated, createResp.StatusCode)

		createRespBody, err := io.ReadAll(createResp.Body)
		require.NoError(t, err)

		var created api.CreateTripResponse
		err = json.Unmarshal(createRespBody, &created)
		require.NoError(t, err)

		require.NotEmpty(t, created.ID)
		require.Equal(t, "draft", string(created.Status))

		publishPayload := api.PublishTripRequest{
			TripID:   created.ID,
			ClientID: created.DriverID,
		}

		publishBody, err := json.Marshal(publishPayload)
		require.NoError(t, err)

		publishReq, err := http.NewRequest(
			http.MethodPost,
			"/trip/publish",
			bytes.NewReader(publishBody),
		)
		require.NoError(t, err)
		publishReq.Header.Set("Content-Type", "application/json")

		publishResp, err := testApp.Test(publishReq, -1)
		require.NoError(t, err)
		defer publishResp.Body.Close()

		require.Equal(t, http.StatusOK, publishResp.StatusCode)

		publishRespBody, err := io.ReadAll(publishResp.Body)
		require.NoError(t, err)

		var published api.PublishTripResponse
		err = json.Unmarshal(publishRespBody, &published)
		require.NoError(t, err)

		require.Equal(t, created.ID, published.TripID)

		getReq, err := http.NewRequest(
			http.MethodGet,
			"/trip/"+created.ID,
			nil,
		)
		require.NoError(t, err)

		getResp, err := testApp.Test(getReq, -1)
		require.NoError(t, err)
		defer getResp.Body.Close()

		require.Equal(t, http.StatusOK, getResp.StatusCode)

		getRespBody, err := io.ReadAll(getResp.Body)
		require.NoError(t, err)

		var got api.CreateTripResponse
		err = json.Unmarshal(getRespBody, &got)
		require.NoError(t, err)

		require.Equal(t, created.ID, got.ID)
		require.Equal(t, created.DriverID, got.DriverID)
		require.Equal(t, "published", string(got.Status))
	})
}
