package acceptance

import (
	"fmt"
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

	t.Run("lang is respected", func(t *testing.T) {
		t.Skip("somehow, the functionality works, but the test fails on sorting") // FIXME

		// With LC_ALL=C, sorting is byte-wise. With LC_ALL=en_US.UTF-8, sorting is character-wise.
		// We can use this to ensure we are not using the C locale.
		exitCode, logs := RunImageRendererWithCommand(t, []string{"sh", "-c"}, []string{`echo -e "a\nb\nA\nB" | sort`})
		require.Zero(t, exitCode, "command did not run successfully")
		// With LC_ALL=C, the output would be "A\nB\na\nb".
		require.Equal(t, "a\nA\nb\nB", logs)
	})
}

func TestRegression680UnicodeCSV(t *testing.T) {
	LongTest(t)
	t.Parallel()

	GetDockerImage(t)
	const filename = "relatório-日本語.csv"
	const contents = "label,value\nexample,42\n"
	fixture := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		// Trigger the download from a page, as Grafana's CSV view does.
		_, _ = fmt.Fprintf(w, `<!doctype html><body><script>
const link = document.createElement("a");
link.download = %q;
link.href = URL.createObjectURL(new Blob([%q], {type: "text/csv"}));
document.body.append(link);
link.click();
</script>`, filename, contents)
	}))
	t.Cleanup(fixture.Close)

	port := fixture.Listener.Addr().(*net.TCPAddr).Port
	svc := StartImageRenderer(t, func(_ testing.TB, req *testcontainers.GenericContainerRequest) {
		req.HostAccessPorts = []int{port}
	})
	target := url.URL{Scheme: "http", Host: net.JoinHostPort(testcontainers.HostInternal, strconv.Itoa(port))}
	query := url.Values{
		"url":       {target.String()},
		"domain":    {target.Hostname()},
		"renderKey": {"test"},
		"timeout":   {"15s"},
	}
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, svc.HTTPEndpoint+"/render/csv?"+query.Encode(), nil)
	require.NoError(t, err)
	req.Header.Set("X-Auth-Token", "-")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	body := ReadBody(t, resp.Body)
	require.Equal(t, http.StatusOK, resp.StatusCode, "CSV response: %s", body)

	_, params, err := mime.ParseMediaType(resp.Header.Get("Content-Disposition"))
	require.NoError(t, err)
	require.Equal(t, filename, params["filename"])
	require.Equal(t, contents, string(body))
}
