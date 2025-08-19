package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/grafana/grafana-image-renderer/pkg/api/middleware"
	"github.com/stretchr/testify/require"
)

func TestTrustedURL(t *testing.T) {
	t.Parallel()

	assertOK := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		url := r.URL.Query().Get("url")
		if url != "" && !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
			panic("invalid URL was passed in: " + url)
		}

		w.WriteHeader(http.StatusOK)
	})

	for name, tc := range map[string]struct {
		Omit           bool
		Value          string
		ExpectedStatus int
	}{
		"no url query":        {Omit: true, ExpectedStatus: http.StatusOK},
		"blank url query":     {Value: "", ExpectedStatus: http.StatusOK},
		"file:// url query":   {Value: "file:///etc/passwd", ExpectedStatus: http.StatusForbidden},
		"chrome:// url query": {Value: "chrome://version", ExpectedStatus: http.StatusForbidden},
		"http:// url query":   {Value: "http://example.com", ExpectedStatus: http.StatusOK},
		"https:// url query":  {Value: "https://example.com", ExpectedStatus: http.StatusOK},
		"invalid URL":         {Value: "://", ExpectedStatus: http.StatusBadRequest},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			rec := httptest.NewRecorder()
			middleware := middleware.TrustedURL(assertOK)
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
			if !tc.Omit {
				q := req.URL.Query()
				q.Set("url", tc.Value)
				req.URL.RawQuery = q.Encode()
			}
			require.NotPanics(t, func() {
				middleware.ServeHTTP(rec, req)
			}, "middleware or handler panicked on request")
			require.Equal(t, tc.ExpectedStatus, rec.Code, "http status code on response")
		})
	}
}
