package contractclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestClientCheckService(t *testing.T) {
	t.Parallel()

	contractService := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/contracts/check-service", r.URL.Path)

		var request struct {
			ClientID    string `json:"client_id"`
			ServiceCode string `json:"service_code"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
		require.Equal(t, "company-123", request.ClientID)
		require.Equal(t, "trip_start", request.ServiceCode)

		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{"allowed":true,"reason":"service_allowed"}`))
		require.NoError(t, err)
	}))
	defer contractService.Close()

	result, err := New(contractService.URL, time.Second, 0).
		CheckService(context.Background(), "company-123", "trip_start")

	require.NoError(t, err)
	require.True(t, result.Allowed)
	require.Equal(t, "service_allowed", result.Reason)
}
