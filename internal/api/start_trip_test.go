package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"job4j.ru/share-trip/internal/api"
	"job4j.ru/share-trip/internal/contractclient"
	"job4j.ru/share-trip/internal/domain"
)

type checkServiceStub struct {
	result contractclient.CheckResult
	err    error
}

func (s checkServiceStub) CheckService(
	_ context.Context,
	_ string,
	_ string,
) (contractclient.CheckResult, error) {
	return s.result, s.err
}

func TestServer_StartTrip(t *testing.T) {
	requireIntegration(t)
	t.Parallel()

	t.Run("success - перевод поездки из published в started", func(t *testing.T) {
		t.Parallel()

		fixture := newTestFixtureWithContracts(checkServiceStub{
			result: contractclient.CheckResult{Allowed: true},
		})
		trip := createPublishedTrip(t, fixture)

		resp := sendStartTrip(t, fixture.app, trip.ID)
		defer closeResponseBody(t, resp.Body)

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var response struct {
			TripID uuid.UUID `json:"tripId"`
		}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&response))
		require.Equal(t, trip.ID, response.TripID)
		requireTripStatus(t, fixture, trip.ID, domain.TripStatusStarted)
	})

	t.Run("denied - статус поездки не меняется", func(t *testing.T) {
		t.Parallel()

		fixture := newTestFixtureWithContracts(checkServiceStub{
			result: contractclient.CheckResult{Allowed: false, Reason: "service is disabled"},
		})
		trip := createPublishedTrip(t, fixture)

		resp := sendStartTrip(t, fixture.app, trip.ID)
		defer closeResponseBody(t, resp.Body)

		require.Equal(t, http.StatusForbidden, resp.StatusCode)
		requireErrorResponse(t, resp, "FORBIDDEN", "forbidden")
		requireTripStatus(t, fixture, trip.ID, domain.TripStatusPublished)
	})

	t.Run("timeout - статус поездки не меняется", func(t *testing.T) {
		t.Parallel()

		contractService := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(50 * time.Millisecond)
		}))
		defer contractService.Close()

		fixture := newTestFixtureWithContracts(contractclient.New(contractService.URL, time.Millisecond, 0))
		trip := createPublishedTrip(t, fixture)

		resp := sendStartTrip(t, fixture.app, trip.ID)
		defer closeResponseBody(t, resp.Body)

		require.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
		requireErrorResponse(t, resp, "CONTRACT_SERVICE_UNAVAILABLE", "cannot verify company permissions")
		requireTripStatus(t, fixture, trip.ID, domain.TripStatusPublished)
	})
}

func createPublishedTrip(t *testing.T, fixture testFixture) api.CreateTripResponse {
	t.Helper()

	trip := createDraftTrip(t, fixture)
	resp := sendMoveTripDraftToPublished(t, fixture.app, api.MoveTripDraftToPublishedRequest{
		TripID: trip.ID.String(),
	})
	defer closeResponseBody(t, resp.Body)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	return trip
}

func sendStartTrip(t *testing.T, app *fiber.App, tripID uuid.UUID) *http.Response {
	t.Helper()

	body, err := json.Marshal(struct {
		TripID uuid.UUID `json:"tripId"`
	}{TripID: tripID})
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, "/trip/start", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	return resp
}

func requireTripStatus(t *testing.T, fixture testFixture, tripID uuid.UUID, want domain.TripStatus) {
	t.Helper()

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("/trip/%s", tripID), nil)
	require.NoError(t, err)

	resp, err := fixture.app.Test(req, -1)
	require.NoError(t, err)
	defer closeResponseBody(t, resp.Body)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var trip api.GetTripByIDResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&trip))
	require.Equal(t, want, trip.Status)
}
