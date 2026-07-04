package api

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"job4j.ru/share-trip/internal/domain"
	"job4j.ru/share-trip/internal/service"
)

type Pinger interface {
	Ping(ctx context.Context) error
}

type TripService interface {
	CreateTrip(ctx context.Context, command service.CreateTripCommand) (domain.Trip, error)
	GetTripByID(ctx context.Context, id string) (domain.Trip, error)
	PublishTrip(ctx context.Context, command service.PublishTripCommand) (string, error)
}

func NewRouter(db Pinger, tripService TripService) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/ready", readyHandler(db))
	mux.HandleFunc("/trip/create", createTripHandler(tripService))
	mux.HandleFunc("/trip/publish", publishTripHandler(tripService))
	mux.HandleFunc("/trip/", getTripByIDHandler(tripService))
	return mux
}

func readyHandler(db Pinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()

		if err := db.Ping(ctx); err != nil {
			slog.Error("database readiness check failed", "error", err)
			http.Error(w, "database unavailable", http.StatusServiceUnavailable)
			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}
}
