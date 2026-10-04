package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

const (
	ResultSuccess          = "success"
	ResultError            = "error"
	ResultValidationError  = "validation_error"
	ResultNotFound         = "not_found"
	ResultForbidden        = "forbidden"
	ResultConflict         = "conflict"
	ResultAlreadyPublished = "already_published"
	ResultInternalError    = "internal_error"
)

const (
	RepositoryOperationTripCreate           = "trip_create"
	RepositoryOperationTripGetForUpdateByID = "trip_get_for_update_by_id"
	RepositoryOperationTripUpdate           = "trip_update"
	RepositoryOperationOutboxEventCreate    = "outbox_event_create"
	RepositoryOperationTripGetByID          = "trip_get_by_id"
)

type Metrics struct {
	HTTPRequestTotal    *prometheus.CounterVec
	HTTPRequestDuration *prometheus.HistogramVec

	TripCreateTotal    *prometheus.CounterVec
	TripCreateDuration *prometheus.HistogramVec

	TripPublishTotal    *prometheus.CounterVec
	TripPublishDuration *prometheus.HistogramVec

	RepositoryQueryTotal    *prometheus.CounterVec
	RepositoryQueryDuration *prometheus.HistogramVec

	ContractRequestTotal    *prometheus.CounterVec
	ContractRequestDuration *prometheus.HistogramVec
	OutboxPending           prometheus.Gauge
	OutboxPublishTotal      *prometheus.CounterVec
	OutboxPublishFailed     prometheus.Counter
}

func New(reg prometheus.Registerer) *Metrics {
	m := &Metrics{
		ContractRequestTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "sharetrip_contract_request_total", Help: "Completed contract checks, including retries within each check",
		}, []string{"result"}),
		ContractRequestDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name: "sharetrip_contract_request_duration_seconds", Help: "Duration of contract checks including retries",
			Buckets: prometheus.DefBuckets,
		}, []string{"result"}),
		OutboxPending: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "sharetrip_outbox_pending_total", Help: "Number of pending outbox events after the latest publisher batch",
		}),
		OutboxPublishTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "sharetrip_outbox_publish_total", Help: "Outbox event publication attempts by result",
		}, []string{"result"}),
		OutboxPublishFailed: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "sharetrip_outbox_publish_failed_total", Help: "Failed outbox event publication attempts",
		}),
		HTTPRequestTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "sharetrip",
				Subsystem: "http",
				Name:      "requests_total",
				Help:      "Total number of HTTP requests",
			},
			[]string{"method", "path", "status"},
		),
		HTTPRequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "sharetrip",
				Subsystem: "http",
				Name:      "request_duration_seconds",
				Help:      "Duration of HTTP requests in seconds",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"method", "path", "status"},
		),
		TripCreateTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "sharetrip",
				Subsystem: "trip",
				Name:      "create_total",
				Help:      "Total number of trip create operations",
			},
			[]string{"result"},
		),
		TripCreateDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "sharetrip",
				Subsystem: "trip",
				Name:      "create_duration_seconds",
				Help:      "Duration of trip create operations in seconds",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"result"},
		),
		TripPublishTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "sharetrip",
				Subsystem: "trip",
				Name:      "publish_total",
				Help:      "Total number of trip publish operations",
			},
			[]string{"result"},
		),
		TripPublishDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "sharetrip",
				Subsystem: "trip",
				Name:      "publish_duration_seconds",
				Help:      "Duration of trip publish operations in seconds",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"result"},
		),
		RepositoryQueryTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "sharetrip",
				Subsystem: "repository",
				Name:      "query_total",
				Help:      "Total number of repository queries",
			},
			[]string{"operation", "result"},
		),
		RepositoryQueryDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "sharetrip",
				Subsystem: "repository",
				Name:      "query_duration_seconds",
				Help:      "Duration of repository queries in seconds",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"operation", "result"},
		),
	}

	reg.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		m.HTTPRequestTotal,
		m.HTTPRequestDuration,
		m.TripCreateTotal,
		m.TripCreateDuration,
		m.TripPublishTotal,
		m.TripPublishDuration,
		m.RepositoryQueryTotal,
		m.RepositoryQueryDuration,
		m.ContractRequestTotal,
		m.ContractRequestDuration,
		m.OutboxPending,
		m.OutboxPublishTotal,
		m.OutboxPublishFailed,
	)

	return m
}
