package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/grafana/grafana-image-renderer/pkg/api"
	"github.com/stretchr/testify/require"
)

func TestGetRenderVersion(t *testing.T) {
	t.Parallel()

	t.Run("version is semver compatible", func(t *testing.T) {
		t.Parallel()

		rec := httptest.NewRecorder()
		handler := api.HandleGetRenderVersion()
		req := httptest.NewRequestWithContext(t.Context(), "GET", "/render/version", nil)
		handler.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)

		var obj struct {
			Version string `json:"version"`
		}
		obj.Version = "!this is very invalid by default!"
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &obj))
		_, err := semver.NewVersion(obj.Version)
		require.NoError(t, err, "version is not semver compatible")
	})
}
