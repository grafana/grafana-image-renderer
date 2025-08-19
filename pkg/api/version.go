package api

import (
	"log/slog"
	"net/http"

	"github.com/grafana/grafana-image-renderer/pkg/chromium"
	"github.com/grafana/grafana-image-renderer/pkg/version"
)

// HandleGetVersion returns the service and browser versions.
func HandleGetVersion(browser *chromium.Browser) http.Handler {
	serviceVersion := version.ServiceVersion()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		version, err := browser.GetVersion(r.Context())
		if err != nil {
			http.Error(w, "failed to get browser version", http.StatusInternalServerError)
			slog.ErrorContext(r.Context(), "failed to get browser version", "error", err)
			return
		}

		_, _ = w.Write([]byte("grafana-image-renderer " + serviceVersion + "\n"))
		_, _ = w.Write([]byte("browser " + version + "\n"))
	})
}
