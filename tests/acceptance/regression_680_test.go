package acceptance

import (
	"io"
	"mime"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
)

// Regression test for: https://github.com/grafana/grafana-image-renderer/issues/680
func TestRegression680(t *testing.T) {
	LongTest(t)
	t.Parallel()

	t.Run("lang is en_US.UTF-8", func(t *testing.T) {
		t.Parallel()

		// The LANG environment variable should be set to en_US.UTF-8, which makes Chromium able to deal with non-ASCII characters properly.

		for _, cmd := range []string{"echo $LANG", "echo $LC_ALL"} {
			t.Run(cmd, func(t *testing.T) {
				t.Parallel()

				exitCode, logs := RunImageRendererWithCommand(t, []string{"sh", "-c"}, []string{cmd})
				require.Zero(t, exitCode, "command %q did not run successfully", cmd)
				require.Equal(t, "en_US.UTF-8", logs, "command %q did not return expected output", cmd)
			})
		}
	})

	t.Run("download preserves a non-ASCII CSV filename", func(t *testing.T) {
		t.Parallel()

		const filename = "résumé-日本語.csv"
		const contents = "city\nMünchen\n"
		page := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = io.WriteString(w, `<html><body><script>
const link = document.createElement("a");
link.href = URL.createObjectURL(new Blob(["city\nMünchen\n"], {type: "text/csv"}));
link.download = "résumé-日本語.csv";
document.body.appendChild(link);
link.click();
</script></body></html>`)
		}))
		t.Cleanup(page.Close)

		_, portString, err := net.SplitHostPort(page.Listener.Addr().String())
		require.NoError(t, err)
		port, err := strconv.Atoi(portString)
		require.NoError(t, err)
		svc := StartImageRenderer(t, func(tb testing.TB, req *testcontainers.GenericContainerRequest) {
			req.HostAccessPorts = []int{port}
		})

		query := url.Values{
			"url":     {"http://host.testcontainers.internal:" + portString},
			"domain":  {"host.testcontainers.internal"},
			"timeout": {"30s"},
		}
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, svc.HTTPEndpoint+"/render/csv?"+query.Encode(), nil)
		require.NoError(t, err)
		req.Header.Set("X-Auth-Token", "-")
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode, "CSV download failed: %s", body)
		_, params, err := mime.ParseMediaType(resp.Header.Get("Content-Disposition"))
		require.NoError(t, err)
		require.Equal(t, filename, params["filename"])
		require.Equal(t, contents, string(body))
	})
}
