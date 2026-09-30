package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"job4j.ru/share-trip/internal/db"
)

type Config struct {
	HTTPAddr             string
	Database             db.Config
	ContractServiceURL   string
	KafkaBrokers         string
	TripEventsTopic      string
	RequestTimeout       time.Duration
	RetryAttempts        int
	KeycloakIssuer       string
	KeycloakClientID     string
	KeycloakClientSecret string
}

func Load() (Config, error) {
	port, err := strconv.Atoi(Env("DB_PORT", "6544"))
	if err != nil {
		return Config{}, fmt.Errorf("parse DB_PORT: %w", err)
	}
	if port < 1 || port > 65535 {
		return Config{}, fmt.Errorf("DB_PORT must be between 1 and 65535")
	}

	timeoutMS, err := strconv.Atoi(Env("CONTRACT_SERVICE_TIMEOUT_MS", "2000"))
	if err != nil {
		return Config{}, fmt.Errorf("parse CONTRACT_SERVICE_TIMEOUT_MS: %w", err)
	}
	if timeoutMS <= 0 || int64(timeoutMS) > (1<<63-1)/int64(time.Millisecond) {
		return Config{}, fmt.Errorf("CONTRACT_SERVICE_TIMEOUT_MS must be positive and fit in time.Duration")
	}

	retryAttempts, err := strconv.Atoi(Env("CONTRACT_SERVICE_RETRY_COUNT", "2"))
	if err != nil {
		return Config{}, fmt.Errorf("parse CONTRACT_SERVICE_RETRY_COUNT: %w", err)
	}
	if retryAttempts < 0 {
		return Config{}, fmt.Errorf("CONTRACT_SERVICE_RETRY_COUNT must not be negative")
	}

	cfg := Config{
		HTTPAddr: Env("HTTP_ADDR", ":8080"),
		Database: db.Config{
			Host:     os.Getenv("DB_HOST"),
			Port:     port,
			User:     os.Getenv("DB_USER"),
			Password: os.Getenv("DB_PASSWORD"),
			Name:     os.Getenv("DB_NAME"),
			SSLMode:  Env("DB_SSLMODE", "disable"),
		},
		ContractServiceURL:   os.Getenv("CONTRACT_SERVICE_BASE_URL"),
		KafkaBrokers:         os.Getenv("KAFKA_BROKERS"),
		TripEventsTopic:      Env("TRIP_EVENTS_TOPIC", "trip.events"),
		RequestTimeout:       time.Duration(timeoutMS) * time.Millisecond,
		RetryAttempts:        retryAttempts,
		KeycloakIssuer:       os.Getenv("KEYCLOAK_ISSUER"),
		KeycloakClientID:     Env("KEYCLOAK_CLIENT_ID", "sharetrip-api"),
		KeycloakClientSecret: os.Getenv("KEYCLOAK_CLIENT_SECRET"),
	}

	if cfg.Database.Host == "" {
		return Config{}, fmt.Errorf("DB_HOST is required")
	}
	if cfg.Database.User == "" {
		return Config{}, fmt.Errorf("DB_USER is required")
	}
	if cfg.Database.Password == "" {
		return Config{}, fmt.Errorf("DB_PASSWORD is required")
	}
	if cfg.Database.Name == "" {
		return Config{}, fmt.Errorf("DB_NAME is required")
	}
	if cfg.ContractServiceURL == "" {
		return Config{}, fmt.Errorf("CONTRACT_SERVICE_BASE_URL is required")
	}
	if cfg.KafkaBrokers == "" {
		return Config{}, fmt.Errorf("KAFKA_BROKERS is required")
	}
	if cfg.KeycloakIssuer == "" {
		return Config{}, fmt.Errorf("KEYCLOAK_ISSUER is required")
	}
	if cfg.KeycloakClientSecret == "" {
		return Config{}, fmt.Errorf("KEYCLOAK_CLIENT_SECRET is required")
	}

	return cfg, nil
}
