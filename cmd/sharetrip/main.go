package main

import (
	"context"
	"errors"
	"log"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/prometheus/client_golang/prometheus"

	"job4j.ru/share-trip/internal/api"
	"job4j.ru/share-trip/internal/config"
	"job4j.ru/share-trip/internal/contractclient"
	"job4j.ru/share-trip/internal/db"
	"job4j.ru/share-trip/internal/events"
	"job4j.ru/share-trip/internal/middleware"
	"job4j.ru/share-trip/internal/observability"
	metrics "job4j.ru/share-trip/internal/observability/metrics"
	"job4j.ru/share-trip/internal/observability/tracing"
	repo "job4j.ru/share-trip/internal/repository"
	"job4j.ru/share-trip/internal/service"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	logger, logFile, err := observability.NewLogger()
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := logFile.Close(); err != nil {
			log.Printf("close log file: %v", err)
		}
	}()

	ctx := context.Background()

	tp, err := tracing.NewProvider(ctx, tracing.Config{
		ServiceName:    "share-trip",
		ServiceVersion: "1.0.0",
		Environment:    "local",
		Endpoint:       "localhost:4319",
	})
	if err != nil {
		logger.Error("init tracing failed", "error", err)
		os.Exit(1)
	}

	defer func() {
		shutdownCtx, cancel := context.WithTimeout(
			context.Background(),
			5*time.Second,
		)
		defer cancel()

		if err := tp.Shutdown(shutdownCtx); err != nil {
			logger.Error("shutdown tracing failed", "error", err)
		}
	}()

	pool, err := db.NewPool(ctx, cfg.Database.DSN())
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	logger.Info("connected to PostgreSQL")

	registry := prometheus.NewRegistry()
	appMetrics := metrics.New(registry)
	tripRepository := repo.NewPostgresTripRepository(pool, appMetrics)
	contracts := contractclient.New(
		cfg.ContractServiceURL,
		cfg.RequestTimeout,
		cfg.RetryAttempts,
	)
	tripService := service.NewTripService(tripRepository, pool, appMetrics, contracts)
	server := api.NewServer(tripService, pool, registry)
	app := fiber.New()
	keycloakAuth := middleware.KeycloakRefreshTokenMiddleware(
		middleware.KeycloakConfig{
			Issuer:       cfg.KeycloakIssuer,
			ClientID:     cfg.KeycloakClientID,
			ClientSecret: cfg.KeycloakClientSecret,
		},
	)
	app.Use(tracing.NewFiberMiddleware())
	app.Use(middleware.Correlation(logger))
	app.Use(middleware.NewHTTPMetricsMiddleware(appMetrics))
	server.RegisterRoutes(app, keycloakAuth, cfg.KeycloakClientID)

	producer := events.NewProducer(
		strings.Split(cfg.KafkaBrokers, ","),
		cfg.TripEventsTopic,
	)
	defer func() {
		if err := producer.Close(); err != nil {
			logger.Error("close Kafka producer", "error", err)
		}
	}()
	publisher := events.NewOutboxPublisher(pool, tripRepository, producer, logger)
	publisherCtx, stopPublisher := context.WithCancel(ctx)
	publisherDone := make(chan struct{})
	go func() {
		defer close(publisherDone)
		if err := publisher.Run(publisherCtx); err != nil && !errors.Is(err, context.Canceled) {
			logger.Error("outbox publisher stopped", "error", err)
		}
	}()
	defer func() {
		stopPublisher()
		<-publisherDone
	}()

	addr := cfg.HTTPAddr

	logger.Info("listening", "address", addr)
	if err := app.Listen(addr); err != nil {
		log.Fatal(err)
	}
}
