package config

import (
	"testing"
	"time"
)

func setValidEnv(t *testing.T) {
	t.Helper()
	for key, value := range map[string]string{
		"HTTP_ADDR": "", "DB_HOST": "postgres", "DB_PORT": "",
		"DB_USER": "postgres", "DB_PASSWORD": "test-password", "DB_NAME": "sharetrip", "DB_SSLMODE": "",
		"CONTRACT_SERVICE_BASE_URL":   "http://contract-service:8080",
		"CONTRACT_SERVICE_TIMEOUT_MS": "", "CONTRACT_SERVICE_RETRY_COUNT": "",
		"KAFKA_BROKERS": "kafka:9092", "TRIP_EVENTS_TOPIC": "",
		"KEYCLOAK_ISSUER":    "http://keycloak/realms/sharetrip",
		"KEYCLOAK_CLIENT_ID": "", "KEYCLOAK_CLIENT_SECRET": "test-client-secret",
	} {
		t.Setenv(key, value)
	}
}

func TestLoadDefaultsAndOverrides(t *testing.T) {
	setValidEnv(t)
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.HTTPAddr != ":8080" || cfg.Database.Port != 6544 || cfg.Database.SSLMode != "disable" ||
		cfg.RequestTimeout != 2*time.Second || cfg.RetryAttempts != 2 ||
		cfg.TripEventsTopic != "trip.events" || cfg.KeycloakClientID != "sharetrip-api" {
		t.Fatal("unexpected defaults")
	}
	t.Setenv("HTTP_ADDR", ":8085")
	t.Setenv("DB_PORT", "5432")
	t.Setenv("DB_SSLMODE", "require")
	t.Setenv("CONTRACT_SERVICE_TIMEOUT_MS", "1500")
	t.Setenv("CONTRACT_SERVICE_RETRY_COUNT", "0")
	t.Setenv("TRIP_EVENTS_TOPIC", "custom-events")
	t.Setenv("KEYCLOAK_CLIENT_ID", "custom-client")
	cfg, err = Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.HTTPAddr != ":8085" || cfg.Database.Port != 5432 || cfg.Database.SSLMode != "require" ||
		cfg.RequestTimeout != 1500*time.Millisecond || cfg.RetryAttempts != 0 ||
		cfg.TripEventsTopic != "custom-events" || cfg.KeycloakClientID != "custom-client" {
		t.Fatal("environment overrides were not applied")
	}
}

func TestLoadRequiresConfiguration(t *testing.T) {
	for _, key := range []string{
		"DB_HOST", "DB_USER", "DB_PASSWORD", "DB_NAME", "CONTRACT_SERVICE_BASE_URL",
		"KAFKA_BROKERS", "KEYCLOAK_ISSUER", "KEYCLOAK_CLIENT_SECRET",
	} {
		t.Run(key, func(t *testing.T) {
			setValidEnv(t)
			t.Setenv(key, "")
			cfg, err := Load()
			if err == nil || err.Error() != key+" is required" || cfg != (Config{}) {
				t.Fatal("expected a configuration error containing only the missing variable name")
			}
		})
	}
}

func TestLoadRejectsInvalidNumbers(t *testing.T) {
	for _, tc := range []struct{ key, value string }{
		{"DB_PORT", "invalid"}, {"DB_PORT", "0"}, {"DB_PORT", "65536"},
		{"CONTRACT_SERVICE_TIMEOUT_MS", "invalid"}, {"CONTRACT_SERVICE_TIMEOUT_MS", "0"},
		{"CONTRACT_SERVICE_TIMEOUT_MS", "-1"}, {"CONTRACT_SERVICE_TIMEOUT_MS", "9223372036854775807"},
		{"CONTRACT_SERVICE_RETRY_COUNT", "invalid"}, {"CONTRACT_SERVICE_RETRY_COUNT", "-1"},
	} {
		t.Run(tc.key+"/"+tc.value, func(t *testing.T) {
			setValidEnv(t)
			t.Setenv(tc.key, tc.value)
			cfg, err := Load()
			if err == nil || cfg != (Config{}) {
				t.Fatal("invalid numeric configuration was accepted")
			}
		})
	}
}
