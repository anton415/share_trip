package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"job4j.ru/go-lang-base/internal/api"
	"job4j.ru/go-lang-base/internal/config"
	"job4j.ru/go-lang-base/internal/db"
)

func main() {
	ctx := context.Background()

	pool, err := db.NewPool(ctx, db.FromEnv().DSN())
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	log.Println("connected to PostgreSQL")

	addr := config.Env("HTTP_ADDR", ":8080")
	server := &http.Server{
		Addr:              addr,
		Handler:           api.NewRouter(pool),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("listening on %s", addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
