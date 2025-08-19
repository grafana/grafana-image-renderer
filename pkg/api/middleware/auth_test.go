package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grafana/grafana-image-renderer/pkg/api/middleware"
	"github.com/stretchr/testify/require"
)

func TestAuthTokenRequired(t *testing.T) {
	t.Parallel()

	alwaysOK := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	for name, tc := range map[string]struct {
		Omit           bool
		Value          string
		ExpectedStatus int
	}{
		"no auth token":      {Omit: true, ExpectedStatus: http.StatusUnauthorized},
		"blank auth token":   {Value: "", ExpectedStatus: http.StatusUnauthorized},
		"wrong auth token":   {Value: "wrong", ExpectedStatus: http.StatusUnauthorized},
		"correct auth token": {Value: "correct", ExpectedStatus: http.StatusOK},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			rec := httptest.NewRecorder()
			middleware := middleware.RequireAuthToken(alwaysOK, "correct")
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
			if !tc.Omit {
				req.Header.Set("X-Auth-Token", tc.Value)
			}
			middleware.ServeHTTP(rec, req)
			require.Equal(t, tc.ExpectedStatus, rec.Code, "http status code on response")
		})
	}
}
