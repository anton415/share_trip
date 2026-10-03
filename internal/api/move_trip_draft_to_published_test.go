package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	"job4j.ru/share-trip/internal/api"
	"job4j.ru/share-trip/internal/domain"
	repo "job4j.ru/share-trip/internal/repository"
	"job4j.ru/share-trip/internal/service"
)

func TestServer_MoveTripDraftToPublished(t *testing.T) {
	requireIntegration(t)
	t.Parallel()

	t.Run("validation error - идентификатор поездки отсутствует", func(t *testing.T) {
		t.Parallel()

		fixture := newTestFixture()
		resp := sendMoveTripDraftToPublished(
			t,
			fixture.app,
			api.MoveTripDraftToPublishedRequest{},
		)
		defer closeResponseBody(t, resp.Body)

		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		requireErrorResponse(t, resp, "VALIDATION_ERROR", "tripId is required")
	})

	t.Run("validation error - идентификатор поездки должен быть UUID", func(t *testing.T) {
		t.Parallel()

		fixture := newTestFixture()
		resp := sendMoveTripDraftToPublished(
			t,
			fixture.app,
			api.MoveTripDraftToPublishedRequest{TripID: "not-a-uuid"},
		)
		defer closeResponseBody(t, resp.Body)

		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		requireErrorResponse(
			t,
			resp,
			"VALIDATION_ERROR",
			"tripId must be a valid UUID",
		)
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
		created := createDraftTrip(t, fixture)

		publishResp := sendMoveTripDraftToPublished(t, fixture.app, api.MoveTripDraftToPublishedRequest{
			TripID: created.ID.String(),
		})
		defer closeResponseBody(t, publishResp.Body)

		require.Equal(t, http.StatusOK, publishResp.StatusCode)
		var outbox domain.OutboxEvent
		err := testPool.QueryRow(testCtx, `
			SELECT id, aggregate_type, aggregate_id, event_type, payload, status
			FROM outbox_events WHERE aggregate_id = $1
		`, created.ID).Scan(
			&outbox.ID, &outbox.AggregateType, &outbox.AggregateID,
			&outbox.EventType, &outbox.Payload, &outbox.Status,
		)
		require.NoError(t, err)
		require.Equal(t, "trip", outbox.AggregateType)
		require.Equal(t, created.ID, outbox.AggregateID)
		require.Equal(t, "TripPublished", outbox.EventType)
		require.Equal(t, "pending", outbox.Status)

		var event domain.TripPublished
		require.NoError(t, json.Unmarshal(outbox.Payload, &event))
		eventID, err := uuid.Parse(event.EventID)
		require.NoError(t, err)
		require.NotEqual(t, uuid.Nil, eventID)
		require.Equal(t, outbox.ID, eventID)
		require.Equal(t, "TripPublished", event.EventType)
		require.Equal(t, created.ID.String(), event.TripID)
		require.Equal(t, fixture.clientID.String(), event.DriverID)
		require.Equal(t, fixture.clientID.String(), event.CompanyID)
		require.False(t, event.OccurredAt.Before(created.CreatedAt))

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
		created := createDraftTrip(t, fixture)

		resp := sendMoveTripDraftToPublished(t, fixture.app, api.MoveTripDraftToPublishedRequest{
			TripID: created.ID.String(),
		}, uuid.NewString())
		defer closeResponseBody(t, resp.Body)

		require.Equal(t, http.StatusForbidden, resp.StatusCode)
		requireErrorResponse(t, resp, "FORBIDDEN", "forbidden")
	})

	t.Run("not found - поездка не существует", func(t *testing.T) {
		t.Parallel()

		fixture := newTestFixture()
		resp := sendMoveTripDraftToPublished(t, fixture.app, api.MoveTripDraftToPublishedRequest{
			TripID: uuid.NewString(),
		})
		defer closeResponseBody(t, resp.Body)

		require.Equal(t, http.StatusNotFound, resp.StatusCode)
		requireErrorResponse(t, resp, "NOT_FOUND", "trip not found")
	})

	t.Run("conflict - поездка не в статусе draft", func(t *testing.T) {
		t.Parallel()

		fixture := newTestFixture()
		created := createDraftTrip(t, fixture)

		_, err := testPool.Exec(
			testCtx,
			`UPDATE trips SET status = 'cancelled' WHERE id = $1`,
			created.ID,
		)
		require.NoError(t, err)

		resp := sendMoveTripDraftToPublished(t, fixture.app, api.MoveTripDraftToPublishedRequest{
			TripID: created.ID.String(),
		})
		defer closeResponseBody(t, resp.Body)

		require.Equal(t, http.StatusConflict, resp.StatusCode)
		requireErrorResponse(t, resp, "CONFLICT", "invalid trip status")
	})

	t.Run("no content - поездка уже published", func(t *testing.T) {
		t.Parallel()

		fixture := newTestFixture()
		created := createDraftTrip(t, fixture)
		payload := api.MoveTripDraftToPublishedRequest{
			TripID: created.ID.String(),
		}

		firstResp := sendMoveTripDraftToPublished(t, fixture.app, payload)
		require.Equal(t, http.StatusOK, firstResp.StatusCode)
		require.NoError(t, firstResp.Body.Close())
		var firstEventID uuid.UUID
		err := testPool.QueryRow(testCtx, `SELECT id FROM outbox_events WHERE aggregate_id = $1`, created.ID).
			Scan(&firstEventID)
		require.NoError(t, err)

		secondResp := sendMoveTripDraftToPublished(t, fixture.app, payload)
		defer closeResponseBody(t, secondResp.Body)

		require.Equal(t, http.StatusNoContent, secondResp.StatusCode)
		body, err := io.ReadAll(secondResp.Body)
		require.NoError(t, err)
		require.Empty(t, body)

		var eventCount, sameEventCount int
		err = testPool.QueryRow(
			testCtx,
			`SELECT COUNT(*), COUNT(*) FILTER (WHERE id = $2)
			 FROM outbox_events WHERE aggregate_id = $1 AND event_type = 'TripPublished'`,
			created.ID,
			firstEventID,
		).Scan(&eventCount, &sameEventCount)
		require.NoError(t, err)
		require.Equal(t, 1, eventCount)
		require.Equal(t, 1, sameEventCount)
	})

	t.Run("outbox error - повтор после отката сохраняет ключ события", func(t *testing.T) {
		t.Parallel()

		fixture := newTestFixture()
		created := createDraftTrip(t, fixture)
		tripRepository := &outboxFailureRepository{
			TripRepository: repo.NewPostgresTripRepository(testPool, fixture.metrics),
		}
		tripService := service.NewTripService(tripRepository, testPool, fixture.metrics, nil)
		tripID, err := tripService.PublishTrip(testCtx, service.PublishTripCommand{
			TripID:   created.ID,
			ClientID: fixture.clientID,
		})
		require.ErrorContains(t, err, "insert outbox event")
		require.Equal(t, uuid.Nil, tripID)

		var status string
		err = testPool.QueryRow(testCtx, `SELECT status FROM trips WHERE id = $1`, created.ID).Scan(&status)
		require.NoError(t, err)
		require.Equal(t, string(domain.TripStatusDraft), status)

		var eventCount int
		err = testPool.QueryRow(testCtx, `SELECT COUNT(*) FROM outbox_events WHERE aggregate_id = $1`, created.ID).
			Scan(&eventCount)
		require.NoError(t, err)
		require.Zero(t, eventCount)

		resp := sendMoveTripDraftToPublished(t, fixture.app, api.MoveTripDraftToPublishedRequest{
			TripID: created.ID.String(),
		})
		defer closeResponseBody(t, resp.Body)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var outboxID uuid.UUID
		var payloadEventID string
		err = testPool.QueryRow(testCtx, `
			SELECT id, payload->>'event_id' FROM outbox_events WHERE aggregate_id = $1
		`, created.ID).Scan(&outboxID, &payloadEventID)
		require.NoError(t, err)
		require.Equal(t, tripRepository.eventID, outboxID)
		require.Equal(t, outboxID.String(), payloadEventID)
	})
}

type outboxFailureRepository struct {
	service.TripRepository
	eventID uuid.UUID
}

func (r *outboxFailureRepository) CreateOutboxEvent(ctx context.Context, tx pgx.Tx, event domain.OutboxEvent) error {
	r.eventID = event.ID
	event.Payload = json.RawMessage(`{`)
	return r.TripRepository.CreateOutboxEvent(ctx, tx, event)
}

func sendMoveTripDraftToPublished(
	t *testing.T,
	app *fiber.App,
	payload api.MoveTripDraftToPublishedRequest,
	subject ...string,
) *http.Response {
	t.Helper()

	body, err := json.Marshal(payload)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, "/trip/publish", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	if len(subject) > 0 {
		req.Header.Set(testSubjectHeader, subject[0])
	}

	resp, err := app.Test(req, -1)
	require.NoError(t, err)

	return resp
}
