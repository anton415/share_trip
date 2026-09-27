package events

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"

	"job4j.ru/share-trip/internal/domain"
	repo "job4j.ru/share-trip/internal/repository"
)

func TestOutboxPublisherRetries(t *testing.T) {
	t.Parallel()
	pool := newPublisherTestPool(t)
	first := seedPublisherEvent(t, pool)
	second := seedPublisherEvent(t, pool)
	var calls []domain.TripPublished
	failed := false
	kafka := tripPublisherFunc(func(_ context.Context, event domain.TripPublished) error {
		calls = append(calls, event)
		if event.EventID == first.EventID && !failed {
			failed = true
			return errors.New("Kafka unavailable")
		}
		return nil
	})
	publisher := NewOutboxPublisher(pool, repo.NewPostgresTripRepository(pool, nil), kafka,
		slog.New(slog.NewTextHandler(io.Discard, nil)))
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	done := make(chan struct{})
	var runErr error
	go func() {
		defer close(done)
		runErr = publisher.Run(ctx)
	}()
	t.Cleanup(func() {
		cancel()
		<-done
	})

	require.Eventually(t, func() bool {
		var sent int
		err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM outbox_events WHERE status = 'sent'`).Scan(&sent)
		return err == nil && sent == 2
	}, 5*time.Second, 20*time.Millisecond)
	cancel()
	<-done
	require.ErrorIs(t, runErr, context.Canceled)
	require.Equal(t, []domain.TripPublished{first, second, first}, calls)

	var attempts int
	var lastError *string
	var sentAt *time.Time
	err := pool.QueryRow(t.Context(), `
		SELECT attempts, last_error, sent_at FROM outbox_events WHERE id = $1
	`, first.EventID).Scan(&attempts, &lastError, &sentAt)
	require.NoError(t, err)
	require.Equal(t, 1, attempts)
	require.NotNil(t, lastError)
	require.Equal(t, "Kafka unavailable", *lastError)
	require.NotNil(t, sentAt)
}

func TestOutboxPublisherMarkSentFailure(t *testing.T) {
	t.Parallel()
	pool := newPublisherTestPool(t)
	first := seedPublisherEvent(t, pool)
	second := seedPublisherEvent(t, pool)
	markErr := errors.New("cannot mark event sent")
	outbox := &markSentFailureRepository{
		outboxRepository: repo.NewPostgresTripRepository(pool, nil),
		failedID:         uuid.MustParse(second.EventID),
		err:              markErr,
	}
	var calls []domain.TripPublished
	kafka := tripPublisherFunc(func(_ context.Context, event domain.TripPublished) error {
		calls = append(calls, event)
		return nil
	})
	publisher := NewOutboxPublisher(pool, outbox, kafka, slog.New(slog.NewTextHandler(io.Discard, nil)))
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()

	require.ErrorIs(t, publisher.publishBatch(ctx), markErr)
	var pending int
	err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM outbox_events WHERE status = 'pending'`).Scan(&pending)
	require.NoError(t, err)
	require.Equal(t, 2, pending)

	outbox.err = nil
	require.NoError(t, publisher.publishBatch(ctx))
	var sent int
	err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM outbox_events WHERE status = 'sent'`).Scan(&sent)
	require.NoError(t, err)
	require.Equal(t, 2, sent)
	require.Equal(t, []domain.TripPublished{first, second, first, second}, calls)
}

type tripPublisherFunc func(context.Context, domain.TripPublished) error

func (f tripPublisherFunc) PublishTripPublished(ctx context.Context, event domain.TripPublished) error {
	return f(ctx, event)
}

type markSentFailureRepository struct {
	outboxRepository
	failedID uuid.UUID
	err      error
}

func (r *markSentFailureRepository) MarkSent(ctx context.Context, tx pgx.Tx, eventID uuid.UUID) error {
	if eventID == r.failedID && r.err != nil {
		return r.err
	}
	return r.outboxRepository.MarkSent(ctx, tx, eventID)
}

func newPublisherTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	testcontainers.SkipIfProviderIsNotHealthy(t)
	container, err := postgres.Run(t.Context(), "postgres:16",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("password"),
		postgres.BasicWaitStrategies(),
	)
	testcontainers.CleanupContainer(t, container)
	require.NoError(t, err)
	dsn, err := container.ConnectionString(t.Context(), "sslmode=disable")
	require.NoError(t, err)
	pool, err := pgxpool.New(t.Context(), dsn)
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	db := stdlib.OpenDBFromPool(pool)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	migrations, err := goose.NewProvider(goose.DialectPostgres, db, os.DirFS("../../migrations"))
	require.NoError(t, err)
	_, err = migrations.Up(t.Context())
	require.NoError(t, err)
	return pool
}

func seedPublisherEvent(t *testing.T, pool *pgxpool.Pool) domain.TripPublished {
	t.Helper()
	event := domain.TripPublished{
		EventID:    uuid.NewString(),
		EventType:  "TripPublished",
		TripID:     uuid.NewString(),
		DriverID:   uuid.NewString(),
		CompanyID:  uuid.NewString(),
		OccurredAt: time.Now().UTC(),
	}
	payload, err := json.Marshal(event)
	require.NoError(t, err)
	_, err = pool.Exec(t.Context(), `
		INSERT INTO outbox_events (id, aggregate_type, aggregate_id, event_type, payload)
		VALUES ($1, 'trip', $2, $3, $4)
	`, event.EventID, event.TripID, event.EventType, payload)
	require.NoError(t, err)
	return event
}
