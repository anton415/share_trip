package domain

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"

	"job4j.ru/share-trip/internal/observability/logctx"
)

type PublishTripRequest struct {
	TripID   uuid.UUID
	ClientID uuid.UUID
}

type PublishTripResponse struct {
	TripID uuid.UUID
}

func (u *TripUsecase) PublishTrip(
	ctx context.Context,
	tx pgx.Tx,
	req PublishTripRequest,
) (*PublishTripResponse, error) {
	ctx, span := otel.Tracer("TripUsecase").
		Start(ctx, "TripUsecase.PublishTrip")
	defer span.End()

	span.SetAttributes(
		attribute.String("trip_id", req.TripID.String()),
		attribute.String("client_id", req.ClientID.String()),
	)

	trip, err := u.tripRepo.GetForUpdateByID(ctx, tx, req.TripID)
	if err != nil {
		return nil, fmt.Errorf("tripRepo.GetForUpdateByID: %w", err)
	}

	if trip.DriverID != req.ClientID {
		return nil, fmt.Errorf(
			"%w: client %s is not driver of trip %s",
			ErrForbidden,
			req.ClientID,
			req.TripID,
		)
	}

	if trip.Status == TripStatusPublished {
		return nil, fmt.Errorf(
			"%w: trip %s",
			ErrTripAlreadyPublished,
			trip.ID,
		)
	}

	if trip.Status != TripStatusDraft {
		return nil, fmt.Errorf(
			"%w: invalid trip status: expected %s, got %s",
			ErrConflict,
			TripStatusDraft,
			trip.Status,
		)
	}

	trip.Status = TripStatusPublished

	updatedTrip, err := u.tripRepo.Update(ctx, tx, trip)
	if err != nil {
		return nil, fmt.Errorf("tripRepo.Update: %w", err)
	}

	eventID := newEventID(updatedTrip.ID, EventTypeTripPublished)
	metadata := propagation.MapCarrier{}
	propagation.TraceContext{}.Inject(ctx, metadata)
	event := TripPublished{
		EventID:       eventID.String(),
		EventType:     EventTypeTripPublished,
		CorrelationID: logctx.CorrelationID(ctx),
		CausationID:   logctx.RequestID(ctx),
		TraceParent:   metadata.Get("traceparent"),
		TripID:        updatedTrip.ID.String(),
		DriverID:      updatedTrip.DriverID.String(),
		CompanyID:     req.ClientID.String(),
		OccurredAt:    time.Now().UTC(),
	}
	payload, err := json.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("marshal trip published payload: %w", err)
	}

	err = u.outboxRepo.CreateOutboxEvent(ctx, tx, OutboxEvent{
		ID:            eventID,
		AggregateType: "trip",
		AggregateID:   updatedTrip.ID,
		EventType:     event.EventType,
		Payload:       payload,
	})
	if err != nil {
		return nil, fmt.Errorf("outboxRepo.CreateOutboxEvent: %w", err)
	}

	return &PublishTripResponse{TripID: updatedTrip.ID}, nil
}

func newEventID(tripID uuid.UUID, eventType string) uuid.UUID {
	return uuid.NewSHA1(tripID, []byte(eventType))
}
