package main

import (
	"context"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/prometheus/client_golang/prometheus"

	"job4j.ru/share-trip/internal/api"
	"job4j.ru/share-trip/internal/api/middleware"
	application "job4j.ru/share-trip/internal/app"
	"job4j.ru/share-trip/internal/config"
	"job4j.ru/share-trip/internal/db"
	observability "job4j.ru/share-trip/internal/observability/metrics"
	"job4j.ru/share-trip/internal/repositories"
	"job4j.ru/share-trip/internal/service"
)

func main() {
	logger, logFile, err := application.NewLogger()
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := logFile.Close(); err != nil {
			log.Printf("close log file: %v", err)
		}
	}()

	ctx := context.Background()

	pool, err := db.NewPool(ctx, db.FromEnv().DSN())
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	logger.Info("connected to PostgreSQL")

	registry := prometheus.NewRegistry()
	appMetrics := observability.New(registry)
	tripRepository := repositories.NewPostgresTripRepository(pool, appMetrics)
	tripService := service.NewTripService(tripRepository, pool, appMetrics)
	server := api.NewServer(tripService, pool, registry)
	app := fiber.New()
	app.Use(middleware.Correlation(logger))
	app.Use(api.NewHTTPMetricsMiddleware(appMetrics))
	server.Route(app)

	addr := config.Env("HTTP_ADDR", ":8080")

	logger.Info("listening", "address", addr)
	if err := app.Listen(addr); err != nil {
		log.Fatal(err)
	}
}
