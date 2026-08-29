package api_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"

	"job4j.ru/share-trip/internal/api"
	"job4j.ru/share-trip/internal/middleware"
)

func TestServer_ProtectedTripRoutes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		claims         *middleware.KeycloakClaims
		expectedStatus int
	}{
		{
			name:           "unauthorized without claims",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "forbidden without client role",
			claims: &middleware.KeycloakClaims{
				Subject: uuid.NewString(),
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "handler reached with client role",
			claims:         claimsWithClientRole(uuid.NewString()),
			expectedStatus: http.StatusBadRequest,
		},
	}

	routes := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{
			name:   "create trip",
			method: http.MethodPost,
			path:   "/trip/create",
			body:   "{",
		},
		{
			name:   "get trip",
			method: http.MethodGet,
			path:   "/trip/not-a-uuid",
		},
	}

	for _, route := range routes {
		route := route
		t.Run(route.name, func(t *testing.T) {
			t.Parallel()

			for _, tt := range tests {
				tt := tt
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()

					app := newRouteTestApp(tt.claims)
					req, err := http.NewRequest(
						route.method,
						route.path,
						strings.NewReader(route.body),
					)
					require.NoError(t, err)
					req.Header.Set("Content-Type", "application/json")

					resp, err := app.Test(req, -1)
					require.NoError(t, err)
					defer closeResponseBody(t, resp.Body)

					require.Equal(t, tt.expectedStatus, resp.StatusCode)
				})
			}
		})
	}
}

func TestServer_CreateTripRejectsInvalidTokenSubject(t *testing.T) {
	t.Parallel()

	app := newRouteTestApp(claimsWithClientRole("not-a-uuid"))
	req, err := http.NewRequest(
		http.MethodPost,
		"/trip/create",
		strings.NewReader("{}"),
	)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	defer closeResponseBody(t, resp.Body)

	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestServer_InfrastructureRoutesBypassKeycloak(t *testing.T) {
	t.Parallel()

	server := api.NewServer(nil, healthyPinger{}, prometheus.NewRegistry())
	app := fiber.New()
	authCalls := 0
	server.RegisterRoutes(
		app,
		func(*fiber.Ctx) error {
			authCalls++
			return fiber.ErrUnauthorized
		},
		testKeycloakClientID,
	)

	for _, path := range []string{"/ready", "/metrics"} {
		req, err := http.NewRequest(http.MethodGet, path, nil)
		require.NoError(t, err)

		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		closeResponseBody(t, resp.Body)
		require.Equal(t, http.StatusOK, resp.StatusCode)
	}

	require.Zero(t, authCalls)
}

func newRouteTestApp(claims *middleware.KeycloakClaims) *fiber.App {
	server := api.NewServer(nil, nil, prometheus.NewRegistry())
	app := fiber.New()
	if claims != nil {
		app.Use(func(c *fiber.Ctx) error {
			c.Locals(middleware.KeycloakClaimsKey, claims)
			return c.Next()
		})
	}
	server.RegisterRoutes(app, passThrough, testKeycloakClientID)

	return app
}

func claimsWithClientRole(subject string) *middleware.KeycloakClaims {
	return &middleware.KeycloakClaims{
		Subject: subject,
		ResourceAccess: map[string]struct {
			Roles []string `json:"roles"`
		}{
			testKeycloakClientID: {Roles: []string{"client"}},
		},
	}
}

type healthyPinger struct{}

func (healthyPinger) Ping(context.Context) error {
	return nil
}
