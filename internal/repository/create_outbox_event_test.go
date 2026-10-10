package repo_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"

	"job4j.ru/share-trip/internal/domain"
	"job4j.ru/share-trip/internal/observability/metrics"
	repo "job4j.ru/share-trip/internal/repository"
)

func TestCreateOutboxEventIdempotency(t *testing.T) {
	t.Parallel()
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
	repository := repo.NewPostgresOutboxRepository(pool, metrics.New(prometheus.NewRegistry()))

	for _, status := range []string{"pending", "sent"} {
		t.Run("duplicate_"+status, func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer cancel()
			event := newOutboxTestEvent()
			tx := beginOutboxTx(t, ctx, pool)
			require.NoError(t, repository.CreateOutboxEvent(ctx, tx, event))
			require.NoError(t, repository.MarkFailed(ctx, tx, event.ID, errors.New("Kafka unavailable")))
			if status == "sent" {
				require.NoError(t, repository.MarkSent(ctx, tx, event.ID))
			}
			require.NoError(t, tx.Commit(ctx))

			var before []byte
			err := pool.QueryRow(ctx, `SELECT to_jsonb(e) FROM outbox_events e WHERE id = $1`, event.ID).Scan(&before)
			require.NoError(t, err)

			event.Payload = json.RawMessage(`{"version":2}`)
			retryTx := beginOutboxTx(t, ctx, pool)
			require.NoError(t, repository.CreateOutboxEvent(ctx, retryTx, event))
			require.NoError(t, retryTx.Commit(ctx))

			var count int
			err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM outbox_events WHERE id = $1`, event.ID).Scan(&count)
			require.NoError(t, err)
			require.Equal(t, 1, count)
			var after []byte
			err = pool.QueryRow(ctx, `SELECT to_jsonb(e) FROM outbox_events e WHERE id = $1`, event.ID).Scan(&after)
			require.NoError(t, err)
			require.JSONEq(t, string(before), string(after))
		})
	}

	t.Run("concurrent_duplicate", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
		defer cancel()
		event := newOutboxTestEvent()
		firstTx := beginOutboxTx(t, ctx, pool)
		require.NoError(t, repository.CreateOutboxEvent(ctx, firstTx, event))
		secondTx := beginOutboxTx(t, ctx, pool)
		secondPID := secondTx.Conn().PgConn().PID()
		duplicate := event
		duplicate.Payload = json.RawMessage(`{"version":2}`)
		result := make(chan error, 1)
		done := make(chan struct{})
		go func() {
			defer close(done)
			err := repository.CreateOutboxEvent(ctx, secondTx, duplicate)
			if err == nil {
				err = secondTx.Commit(ctx)
			}
			result <- err
		}()
		t.Cleanup(func() {
			cancel()
			<-done
		})

		// Keep the first insert uncommitted until the second waits on its lock.
		require.Eventually(t, func() bool {
			var waiting bool
			err := pool.QueryRow(ctx, `
				SELECT EXISTS (
					SELECT 1 FROM pg_stat_activity WHERE pid = $1 AND wait_event_type = 'Lock'
				)
			`, secondPID).Scan(&waiting)
			return err == nil && waiting
		}, 5*time.Second, 10*time.Millisecond)
		require.NoError(t, firstTx.Commit(ctx))
		select {
		case err := <-result:
			require.NoError(t, err)
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		}

		var count int
		err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM outbox_events WHERE id = $1`, event.ID).Scan(&count)
		require.NoError(t, err)
		require.Equal(t, 1, count)
		var payload []byte
		err = pool.QueryRow(ctx, `SELECT payload FROM outbox_events WHERE id = $1`, event.ID).Scan(&payload)
		require.NoError(t, err)
		require.JSONEq(t, string(event.Payload), string(payload))
	})
}

func newOutboxTestEvent() domain.OutboxEvent {
	tripID := uuid.New()
	return domain.OutboxEvent{
		ID:            uuid.NewSHA1(tripID, []byte(domain.EventTypeTripPublished)),
		AggregateType: "trip",
		AggregateID:   tripID,
		EventType:     domain.EventTypeTripPublished,
		Payload:       json.RawMessage(`{"version":1}`),
	}
}
