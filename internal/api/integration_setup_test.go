package api_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"

	"job4j.ru/share-trip/internal/api"
	"job4j.ru/share-trip/internal/middleware"
	observability "job4j.ru/share-trip/internal/observability/metrics"
	"job4j.ru/share-trip/internal/repository"
	"job4j.ru/share-trip/internal/service"
)

const (
	testKeycloakClientID = "sharetrip-api"
	testSubjectHeader    = "X-Test-Subject"
)

var (
	testCtx              context.Context
	testDB               *sql.DB
	testPool             *pgxpool.Pool
	testContainer        *postgres.PostgresContainer
	integrationSetupOnce sync.Once
	integrationSetupErr  error
)

func TestMain(m *testing.M) {
	testCtx = context.Background()
	code := m.Run()
	cleanupIntegration()
	os.Exit(code)
}

func requireIntegration(t *testing.T) {
	t.Helper()
	testcontainers.SkipIfProviderIsNotHealthy(t)

	integrationSetupOnce.Do(func() {
		integrationSetupErr = setupIntegration()
	})
	if integrationSetupErr != nil {
		t.Fatalf("setup integration tests: %v", integrationSetupErr)
	}
}

func setupIntegration() error {
	var err error

	testContainer, err = postgres.Run(
		testCtx,
		"postgres:16",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("password"),
	)
	if err != nil {
		return fmt.Errorf("start postgres container: %w", err)
	}

	dsn, err := testContainer.ConnectionString(
		testCtx,
		"sslmode=disable",
	)
	if err != nil {
		return fmt.Errorf("get connection string: %w", err)
	}

	testDB, err = sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("open sql db: %w", err)
	}

	if err = waitReady(testDB); err != nil {
		return err
	}

	if err = goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}

	if err = goose.Up(testDB, "../../migrations"); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	testPool, err = pgxpool.New(testCtx, dsn)
	if err != nil {
		return fmt.Errorf("create pgx pool: %w", err)
	}

	return nil
}

type testFixture struct {
	app      *fiber.App
	clientID uuid.UUID
	metrics  *observability.Metrics
}

func newTestFixture() testFixture {
	registry := prometheus.NewRegistry()
	appMetrics := observability.New(registry)

	tripRepository := repo.NewPostgresTripRepository(testPool, appMetrics)
	tripService := service.NewTripService(tripRepository, testPool, appMetrics, nil)

	server := api.NewServer(tripService, testPool, registry)
	clientID := uuid.New()

	app := fiber.New()
	app.Use(middleware.NewHTTPMetricsMiddleware(appMetrics))
	app.Use(func(c *fiber.Ctx) error {
		subject := c.Get(testSubjectHeader)
		if subject == "" {
			subject = clientID.String()
		}

		c.Locals(middleware.KeycloakClaimsKey, &middleware.KeycloakClaims{
			Subject: subject,
			ResourceAccess: map[string]struct {
				Roles []string `json:"roles"`
			}{
				testKeycloakClientID: {Roles: []string{"client"}},
			},
		})

		return c.Next()
	})
	server.RegisterRoutes(app, passThrough, testKeycloakClientID)

	return testFixture{
		app:      app,
		clientID: clientID,
		metrics:  appMetrics,
	}
}

func passThrough(c *fiber.Ctx) error {
	return c.Next()
}

func cleanupIntegration() {
	if testPool != nil {
		testPool.Close()
	}
	if testDB != nil {
		_ = testDB.Close()
	}
	if testContainer != nil {
		_ = testContainer.Terminate(testCtx)
	}
}

func waitReady(db *sql.DB) error {
	deadline := time.Now().Add(30 * time.Second)
	var pingErr error

	for time.Now().Before(deadline) {
		ctx, cancel := context.WithTimeout(
			context.Background(),
			2*time.Second,
		)
		pingErr = db.PingContext(ctx)
		cancel()

		if pingErr == nil {
			return nil
		}

		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("database is not ready after timeout: %w", pingErr)
}
