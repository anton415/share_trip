package api_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestServer_Ready(t *testing.T) {
	requireIntegration(t)
	t.Parallel()

	fixture := newTestFixture()
	req, err := http.NewRequest(http.MethodGet, "/ready", nil)
	require.NoError(t, err)

	resp, err := fixture.app.Test(req, -1)
	require.NoError(t, err)
	defer closeResponseBody(t, resp.Body)

	require.Equal(t, http.StatusOK, resp.StatusCode)
}
