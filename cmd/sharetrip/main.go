package main

import (
	"context"
	"log"

	"github.com/gofiber/fiber/v2"

	"job4j.ru/share-trip/internal/api"
	"job4j.ru/share-trip/internal/config"
	"job4j.ru/share-trip/internal/db"
	"job4j.ru/share-trip/internal/repositories"
	"job4j.ru/share-trip/internal/service"
)

func main() {
	ctx := context.Background()

	pool, err := db.NewPool(ctx, db.FromEnv().DSN())
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	log.Println("connected to PostgreSQL")

	tripRepository := repositories.NewPostgresTripRepository(pool)
	tripService := service.NewTripService(tripRepository, pool)
	server := api.NewServer(tripService, pool)
	app := fiber.New()
	server.Route(app.Group(""))

	addr := config.Env("HTTP_ADDR", ":8080")

	log.Printf("listening on %s", addr)
	if err := app.Listen(addr); err != nil {
		log.Fatal(err)
	}
}
