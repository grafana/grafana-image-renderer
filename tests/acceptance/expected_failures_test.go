package acceptance

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAuthTokenFailures(t *testing.T) {
	LongTest(t)
	t.Parallel()

	svc := StartImageRenderer(t)

	for name, token := range map[string]*string{
		"no token":         nil,
		"empty token":      ptr(""),
		"whitespace token": ptr(" \t"),
		"wrong token":      ptr("wrong"),
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, svc.HTTPEndpoint+"/render?url=http://localhost:8081/", nil)
			require.NoError(t, err, "could not construct HTTP request to /render")
			req.Header.Set("Accept", "*/*")
			if token != nil {
				req.Header.Set("X-Auth-Token", *token)
			}
			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err, "could not send HTTP request to /render")
			require.Equal(t, http.StatusUnauthorized, resp.StatusCode, "expected 401 Unauthorized response")
		})
	}
}

func TestRejectedURLs(t *testing.T) {
	LongTest(t)
	t.Parallel()

	svc := StartImageRenderer(t)

	for name, url := range map[string]string{
		"file scheme":   "file:///etc/passwd",
		"chrome scheme": "chrome://version",
		"ftp scheme":    "ftp://localhost:21/",
		"socket scheme": "socket://localhost:8081/",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, svc.HTTPEndpoint+"/render?url="+url, nil)
			require.NoError(t, err, "could not construct HTTP request to /render")
			req.Header.Set("Accept", "*/*")
			req.Header.Set("X-Auth-Token", "-")

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err, "could not send HTTP request to /render")
			require.Equal(t, http.StatusForbidden, resp.StatusCode, "expected 403 Forbidden response")
		})
	}
}

func TestInvalidQueryParameters(t *testing.T) {
	LongTest(t)
	t.Parallel()

	svc := StartImageRenderer(t)

	for name, params := range map[string]map[string]string{
		"no URL query": {},
		"empty URL":    {"url": ""},
		// "text width":       {"url": "http://localhost:8081/", "width": "invalid"}, // node.js doesn't do this
		// "text height":      {"url": "http://localhost:8081/", "height": "invalid"}, // node.js doesn't do this
		// "unknown encoding": {"url": "http://localhost:8081/", "encoding": "invalid"}, // node.js doesn't do this
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			reqURL, err := url.Parse(svc.HTTPEndpoint + "/render")
			require.NoError(t, err, "could not parse /render URL")
			query := reqURL.Query()
			for k, v := range params {
				query.Set(k, v)
			}
			reqURL.RawQuery = query.Encode()

			req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, reqURL.String(), nil)
			require.NoError(t, err, "could not construct HTTP request to /render")
			req.Header.Set("Accept", "*/*")
			req.Header.Set("X-Auth-Token", "-")

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err, "could not send HTTP request to /render")
			require.Equal(t, http.StatusBadRequest, resp.StatusCode, "expected 400 Bad Request response")
		})
	}
}

func TestBrowserTimeout(t *testing.T) {
	t.Parallel()

	t.Run("if the browser readiness timeout is exceeded, it returns a 408 status code", func(t *testing.T) {
		t.Parallel()

		// extremely low that no request could ever be server in the alloted time
		svc := StartImageRenderer(t, WithEnv("BROWSER_READINESS_TIMEOUT", "1ns"))

		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, svc.HTTPEndpoint+"/render?url=http://localhost:8081/&encoding=pdf", nil)
		require.NoError(t, err, "could not construct HTTP request to /render")
		req.Header.Set("Accept", "application/pdf")
		req.Header.Set("X-Auth-Token", "-")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err, "could not make HTTP request to /render")
		require.Equal(t, http.StatusRequestTimeout, resp.StatusCode, "expected HTTP 408 Request Timeout from /render")
	})
}
