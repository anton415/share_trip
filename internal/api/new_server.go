package api

import "github.com/prometheus/client_golang/prometheus"

func NewServer(
	trips TripService,
	db Pinger,
	registry prometheus.Gatherer,
) *Server {
	return &Server{
		trips:    trips,
		db:       db,
		registry: registry,
	}
}
