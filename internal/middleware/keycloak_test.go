package middleware

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/require"
)

func TestRefreshAccessToken(t *testing.T) {
	t.Parallel()

	type capturedRequest struct {
		method      string
		path        string
		contentType string
		form        url.Values
		parseErr    error
	}

	requests := make(chan capturedRequest, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parseErr := r.ParseForm()
		requests <- capturedRequest{
			method:      r.Method,
			path:        r.URL.Path,
			contentType: r.Header.Get("Content-Type"),
			form:        r.PostForm,
			parseErr:    parseErr,
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"access_token":"access-token"}`)
	}))
	defer server.Close()

	token, err := refreshAccessToken(
		context.Background(),
		server.Client(),
		KeycloakConfig{
			Issuer:       server.URL + "/realms/sharetrip/",
			ClientID:     "sharetrip-client",
			ClientSecret: "client-secret",
		},
		"refresh-token",
	)
	require.NoError(t, err)
	require.Equal(t, "access-token", token.AccessToken)

	request := <-requests
	require.NoError(t, request.parseErr)
	require.Equal(t, http.MethodPost, request.method)
	require.Equal(t, "/realms/sharetrip/protocol/openid-connect/token", request.path)
	require.Equal(t, "application/x-www-form-urlencoded", request.contentType)
	require.Equal(t, url.Values{
		"client_id":     {"sharetrip-client"},
		"client_secret": {"client-secret"},
		"grant_type":    {"refresh_token"},
		"refresh_token": {"refresh-token"},
	}, request.form)
}

func TestKeycloakRefreshTokenMiddlewareMissingHeader(t *testing.T) {
	t.Parallel()

	app := fiber.New()
	app.Use(KeycloakRefreshTokenMiddleware(KeycloakConfig{}))
	app.Get("/protected", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusNoContent)
	})

	req, err := http.NewRequest(http.MethodGet, "/protected", nil)
	require.NoError(t, err)

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, resp.Body.Close())
	}()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
	require.Contains(t, string(body), "missing refresh token")
}

func TestKeycloakRefreshTokenMiddlewareRejectsInvalidTokens(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		keycloakStatus   int
		keycloakResponse string
		wantMessage      string
	}{
		{
			name:             "non-200 token response",
			keycloakStatus:   http.StatusBadRequest,
			keycloakResponse: `{"error":"invalid_grant"}`,
			wantMessage:      "invalid refresh token",
		},
		{
			name:             "malformed access token",
			keycloakStatus:   http.StatusOK,
			keycloakResponse: `{"access_token":"not-a-jwt"}`,
			wantMessage:      "invalid access token",
		},
		{
			name:             "missing access token",
			keycloakStatus:   http.StatusOK,
			keycloakResponse: `{}`,
			wantMessage:      "invalid refresh token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.keycloakStatus)
				_, _ = io.WriteString(w, tt.keycloakResponse)
			}))
			defer server.Close()

			app := fiber.New()
			app.Use(KeycloakRefreshTokenMiddleware(KeycloakConfig{
				Issuer:     server.URL + "/realms/sharetrip",
				ClientID:   "sharetrip-client",
				HTTPClient: server.Client(),
			}))
			app.Get("/protected", func(c *fiber.Ctx) error {
				return c.SendStatus(fiber.StatusNoContent)
			})

			req, err := http.NewRequest(http.MethodGet, "/protected", nil)
			require.NoError(t, err)
			req.Header.Set(RefreshTokenHeader, "refresh-token")

			resp, err := app.Test(req)
			require.NoError(t, err)
			defer func() {
				require.NoError(t, resp.Body.Close())
			}()

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			require.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
			require.Contains(t, string(body), tt.wantMessage)
		})
	}
}

func TestKeycloakRefreshTokenMiddlewareStoresClaims(t *testing.T) {
	t.Parallel()

	wantClaims := &KeycloakClaims{
		Subject:           "user-id",
		PreferredUsername: "petr",
		Email:             "petr@example.com",
		AuthorizedParty:   "sharetrip-client",
		ResourceAccess: map[string]struct {
			Roles []string `json:"roles"`
		}{
			"sharetrip-client": {Roles: []string{"user", "admin"}},
		},
	}
	accessToken := makeTestAccessToken(t, wantClaims)
	tokenResponse, err := json.Marshal(keycloakTokenResponse{AccessToken: accessToken})
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(tokenResponse)
	}))
	defer server.Close()

	observedClaims := make(chan *KeycloakClaims, 1)
	app := fiber.New()
	app.Use(KeycloakRefreshTokenMiddleware(KeycloakConfig{
		Issuer:     server.URL + "/realms/sharetrip",
		ClientID:   "sharetrip-client",
		HTTPClient: server.Client(),
	}))
	app.Get("/protected", func(c *fiber.Ctx) error {
		claims, err := ClaimsFromContext(c)
		if err != nil {
			return err
		}

		observedClaims <- claims
		return c.SendStatus(fiber.StatusNoContent)
	})

	req, err := http.NewRequest(http.MethodGet, "/protected", nil)
	require.NoError(t, err)
	req.Header.Set(RefreshTokenHeader, "refresh-token")

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, resp.Body.Close())
	}()
	require.Equal(t, fiber.StatusNoContent, resp.StatusCode)
	require.Equal(t, wantClaims, <-observedClaims)
}

func TestRequireClientRole(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		claims      *KeycloakClaims
		wantStatus  int
		wantReached bool
	}{
		{
			name:       "missing claims",
			wantStatus: fiber.StatusUnauthorized,
		},
		{
			name: "missing role",
			claims: &KeycloakClaims{
				Subject: "user-id",
				ResourceAccess: map[string]struct {
					Roles []string `json:"roles"`
				}{
					"sharetrip-client": {Roles: []string{"user"}},
				},
			},
			wantStatus: fiber.StatusForbidden,
		},
		{
			name: "matching role",
			claims: &KeycloakClaims{
				Subject: "user-id",
				ResourceAccess: map[string]struct {
					Roles []string `json:"roles"`
				}{
					"sharetrip-client": {Roles: []string{"user", "admin"}},
				},
			},
			wantStatus:  fiber.StatusNoContent,
			wantReached: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			app := fiber.New()
			if tt.claims != nil {
				app.Use(func(c *fiber.Ctx) error {
					c.Locals(KeycloakClaimsKey, tt.claims)
					return c.Next()
				})
			}
			app.Use(RequireClientRole("sharetrip-client", "admin"))

			reached := false
			app.Get("/protected", func(c *fiber.Ctx) error {
				reached = true
				return c.SendStatus(fiber.StatusNoContent)
			})

			req, err := http.NewRequest(http.MethodGet, "/protected", nil)
			require.NoError(t, err)

			resp, err := app.Test(req)
			require.NoError(t, err)
			defer func() {
				require.NoError(t, resp.Body.Close())
			}()
			require.Equal(t, tt.wantStatus, resp.StatusCode)
			require.Equal(t, tt.wantReached, reached)
		})
	}
}

func TestParseAccessTokenClaims(t *testing.T) {
	t.Parallel()

	wantClaims := &KeycloakClaims{
		Subject:           "user-id",
		PreferredUsername: "petr",
		AuthorizedParty:   "sharetrip-client",
	}

	claims, err := parseAccessTokenClaims(makeTestAccessToken(t, wantClaims))
	require.NoError(t, err)
	require.Equal(t, wantClaims, claims)

	tests := []struct {
		name       string
		token      string
		wantErrMsg string
	}{
		{
			name:       "wrong number of parts",
			token:      "two.parts",
			wantErrMsg: "jwt must contain three parts",
		},
		{
			name:       "invalid base64 payload",
			token:      "header.%.signature",
			wantErrMsg: "illegal base64 data",
		},
		{
			name:       "invalid JSON payload",
			token:      "header." + base64.RawURLEncoding.EncodeToString([]byte("not-json")) + ".signature",
			wantErrMsg: "invalid character",
		},
		{
			name:       "missing subject",
			token:      "header." + base64.RawURLEncoding.EncodeToString([]byte(`{"preferred_username":"petr"}`)) + ".signature",
			wantErrMsg: "jwt does not contain sub claim",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			claims, err := parseAccessTokenClaims(tt.token)
			require.ErrorContains(t, err, tt.wantErrMsg)
			require.Nil(t, claims)
		})
	}
}

func TestKeycloakClaimsHasClientRole(t *testing.T) {
	t.Parallel()

	claims := KeycloakClaims{
		ResourceAccess: map[string]struct {
			Roles []string `json:"roles"`
		}{
			"sharetrip-client": {Roles: []string{"user", "admin"}},
		},
	}

	require.True(t, claims.HasClientRole("sharetrip-client", "admin"))
	require.False(t, claims.HasClientRole("sharetrip-client", "owner"))
	require.False(t, claims.HasClientRole("another-client", "admin"))
}

func makeTestAccessToken(t *testing.T, claims *KeycloakClaims) string {
	t.Helper()

	payload, err := json.Marshal(claims)
	require.NoError(t, err)

	return "header." + base64.RawURLEncoding.EncodeToString(payload) + ".signature"
}
