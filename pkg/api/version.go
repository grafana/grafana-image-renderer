package api

import (
	"log/slog"
	"net/http"

	"github.com/grafana/grafana-image-renderer/pkg/service"
)

// HandleGetVersion returns the service and browser versions.
func HandleGetVersion(versions *service.VersionService, browser *service.BrowserService) http.Handler {
	serviceVersion := versions.GetPrettyVersion()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tracer := tracer(r.Context())
		ctx, span := tracer.Start(r.Context(), "HandleGetVersion")
		defer span.End()

		version, err := browser.GetVersion(ctx)
		if err != nil {
			http.Error(w, "failed to get browser version", http.StatusInternalServerError)
			slog.ErrorContext(ctx, "failed to get browser version", "error", err)
			return
		}

		_, _ = w.Write([]byte("grafana-image-renderer " + serviceVersion + "\n"))
		_, _ = w.Write([]byte("browser " + version + "\n"))
	})
}

// HandleGetRenderVersion returns the service and browser versions.
func HandleGetRenderVersion(versions *service.VersionService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tracer := tracer(r.Context())
		_, span := tracer.Start(r.Context(), "HandleGetRenderVersion")
		defer span.End()

		response := `{"version": "` + versions.GetRenderVersion() + `"}`
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(response))
	})
}
