package main

import (
	"context"
	"log"
	"net/http"
	"time"

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
	tripService := service.NewTripService(tripRepository)

	addr := config.Env("HTTP_ADDR", ":8080")
	server := &http.Server{
		Addr:              addr,
		Handler:           api.NewRouter(pool, tripService),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("listening on %s", addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
