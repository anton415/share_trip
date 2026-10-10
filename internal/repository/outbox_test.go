package repo_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

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

func TestOutboxRepository(t *testing.T) {
	t.Parallel()
	testcontainers.SkipIfProviderIsNotHealthy(t)
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()

	container, err := postgres.Run(ctx, "postgres:16",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("password"),
		postgres.BasicWaitStrategies(),
	)
	testcontainers.CleanupContainer(t, container)
	require.NoError(t, err)
	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	db := stdlib.OpenDBFromPool(pool)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	migrations, err := goose.NewProvider(goose.DialectPostgres, db, os.DirFS("../../migrations"))
	require.NoError(t, err)
	_, err = migrations.UpTo(ctx, 4)
	require.NoError(t, err)
	legacyID := "00000000-0000-4000-8000-000000000001"
	legacyTripID := "00000000-0000-4000-8000-000000000002"
	legacyPayload := `{"trip_id":"00000000-0000-4000-8000-000000000002"}`
	_, err = pool.Exec(ctx, `
		INSERT INTO outbox_event (id, event_name, aggregate_id, payload)
		VALUES ($1, 'trip_published', $2, $3)
	`, legacyID, legacyTripID, legacyPayload)
	require.NoError(t, err)
	_, err = migrations.Up(ctx)
	require.NoError(t, err)
	var legacy domain.OutboxEvent
	err = pool.QueryRow(ctx, `
		SELECT payload, status, attempts, last_error, sent_at FROM outbox_events WHERE id = $1
	`, legacyID).Scan(&legacy.Payload, &legacy.Status, &legacy.Attempts, &legacy.LastError, &legacy.SentAt)
	require.NoError(t, err)
	require.JSONEq(t, legacyPayload, string(legacy.Payload))
	require.Equal(t, "failed", legacy.Status)
	require.Zero(t, legacy.Attempts)
	require.NotNil(t, legacy.LastError)
	require.Contains(t, *legacy.LastError, "original Kafka event_id and delivery status are unknown")
	require.Nil(t, legacy.SentAt)
	repository := repo.NewPostgresOutboxRepository(pool, nil)

	_, err = pool.Exec(ctx, `
		INSERT INTO outbox_events (aggregate_type, aggregate_id, event_type, payload, created_at)
		SELECT 'trip', gen_random_uuid(), 'TripPublished', '{}',
		       '2026-01-01'::timestamptz + n * interval '1 second'
		FROM generate_series(101, 1, -1) AS n;
		INSERT INTO outbox_events (aggregate_type, aggregate_id, event_type, payload, status, created_at, sent_at)
		VALUES ('trip', gen_random_uuid(), 'TripPublished', '{}', 'sent', '2025-01-01', '2025-01-01');
	`)
	require.NoError(t, err)

	pending, err := repository.CountPending(ctx)
	require.NoError(t, err)
	require.Equal(t, 101, pending)
	firstTx := beginOutboxTx(t, ctx, pool)
	firstBatch, err := repository.LockPending(ctx, firstTx)
	require.NoError(t, err)
	require.Len(t, firstBatch, 100)
	for i, event := range firstBatch {
		require.Equal(t, "pending", event.Status)
		require.Nil(t, event.LastError)
		require.Nil(t, event.SentAt)
		if i > 0 {
			require.True(t, firstBatch[i-1].CreatedAt.Before(event.CreatedAt))
		}
	}

	secondTx := beginOutboxTx(t, ctx, pool)
	secondBatch, err := repository.LockPending(ctx, secondTx)
	require.NoError(t, err)
	require.Len(t, secondBatch, 1)
	require.True(t, secondBatch[0].CreatedAt.After(firstBatch[99].CreatedAt))
	for _, event := range firstBatch {
		require.NotEqual(t, event.ID, secondBatch[0].ID)
	}

	failedID, sentID := firstBatch[0].ID, firstBatch[1].ID
	require.NoError(t, repository.MarkFailed(ctx, firstTx, failedID, errors.New("Kafka unavailable")))
	require.NoError(t, repository.MarkFailed(ctx, firstTx, failedID, errors.New("Kafka timeout")))
	require.NoError(t, repository.MarkSent(ctx, firstTx, sentID))
	require.NoError(t, firstTx.Commit(ctx))
	require.NoError(t, secondTx.Rollback(ctx))

	var sentStatus string
	var sentAt *time.Time
	err = pool.QueryRow(ctx, `SELECT status, sent_at FROM outbox_events WHERE id = $1`, sentID).
		Scan(&sentStatus, &sentAt)
	require.NoError(t, err)
	require.Equal(t, "sent", sentStatus)
	require.NotNil(t, sentAt)

	retryTx := beginOutboxTx(t, ctx, pool)
	retryBatch, err := repository.LockPending(ctx, retryTx)
	require.NoError(t, err)
	require.Len(t, retryBatch, 100)
	var retried *domain.OutboxEvent
	for i := range retryBatch {
		require.NotEqual(t, sentID, retryBatch[i].ID)
		if retryBatch[i].ID == failedID {
			retried = &retryBatch[i]
		}
	}
	require.NotNil(t, retried)
	require.Equal(t, "pending", retried.Status)
	require.Equal(t, 2, retried.Attempts)
	require.NotNil(t, retried.LastError)
	require.Equal(t, "Kafka timeout", *retried.LastError)
	require.Equal(t, firstBatch[0].Payload, retried.Payload)

	require.NoError(t, repository.MarkSent(ctx, retryTx, failedID))
	require.NoError(t, repository.MarkFailed(ctx, retryTx, secondBatch[0].ID, errors.New("rolled back")))
	require.NoError(t, retryTx.Rollback(ctx))
	var status string
	err = pool.QueryRow(ctx, `SELECT status, sent_at FROM outbox_events WHERE id = $1`, failedID).
		Scan(&status, &sentAt)
	require.NoError(t, err)
	require.Equal(t, "pending", status)
	require.Nil(t, sentAt)
	var attempts int
	var lastError *string
	err = pool.QueryRow(ctx, `SELECT attempts, last_error FROM outbox_events WHERE id = $1`, secondBatch[0].ID).
		Scan(&attempts, &lastError)
	require.NoError(t, err)
	require.Zero(t, attempts)
	require.Nil(t, lastError)
}

func beginOutboxTx(t *testing.T, ctx context.Context, pool *pgxpool.Pool) pgx.Tx {
	t.Helper()
	tx, err := pool.Begin(ctx)
	require.NoError(t, err)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			t.Errorf("rollback outbox transaction: %v", err)
		}
	})
	return tx
}
