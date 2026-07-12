package main

import (
	"context"
	"github.com/gofiber/fiber/v2"
	application "job4j.ru/share-trip/internal/app"
	"log"

	"job4j.ru/share-trip/internal/api"
	"job4j.ru/share-trip/internal/api/middleware"
	"job4j.ru/share-trip/internal/config"
	"job4j.ru/share-trip/internal/db"
	"job4j.ru/share-trip/internal/repositories"
	"job4j.ru/share-trip/internal/service"
)

func main() {
	logger, logFile, err := application.NewLogger()
	if err != nil {
		log.Fatal(err)
	}
	defer logFile.Close()

	ctx := context.Background()

	pool, err := db.NewPool(ctx, db.FromEnv().DSN())
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	logger.Info("connected to PostgreSQL")

	tripRepository := repositories.NewPostgresTripRepository(pool)
	tripService := service.NewTripService(tripRepository, pool)
	server := api.NewServer(tripService, pool)
	app := fiber.New()
	app.Use(middleware.Correlation(logger))
	server.Route(app.Group(""))

	addr := config.Env("HTTP_ADDR", ":8080")

	logger.Info("listening", "address", addr)
	if err := app.Listen(addr); err != nil {
		log.Fatal(err)
	}
}
