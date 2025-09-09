package api

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/grafana/grafana-image-renderer/pkg/service"
)

func HandlePostRenderCSV(browser *service.BrowserService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		url := r.URL.Query().Get("url")
		if url == "" {
			http.Error(w, "missing 'url' query parameter", http.StatusBadRequest)
			return
		}
		if encoding := r.URL.Query().Get("encoding"); encoding != "" && encoding != "csv" {
			http.Error(w, "invalid 'encoding' query parameter: must be 'csv' or empty/missing", http.StatusBadRequest)
			return
		}
		ctx := r.Context()
		if timeout := r.URL.Query().Get("timeout"); timeout != "" {
			if regexpOnlyNumbers.MatchString(timeout) {
				seconds, err := strconv.Atoi(timeout)
				if err != nil {
					http.Error(w, fmt.Sprintf("invalid 'timeout' query parameter: %v", err), http.StatusBadRequest)
					return
				}
				timeoutCtx, cancelTimeout := context.WithTimeout(r.Context(), time.Duration(seconds)*time.Second)
				defer cancelTimeout()
				ctx = timeoutCtx
			} else {
				timeout, err := time.ParseDuration(timeout)
				if err != nil {
					http.Error(w, fmt.Sprintf("invalid 'timeout' query parameter: %v", err), http.StatusBadRequest)
					return
				}
				timeoutCtx, cancelTimeout := context.WithTimeout(r.Context(), timeout)
				defer cancelTimeout()
				ctx = timeoutCtx
			}
		}
		renderKey := r.URL.Query().Get("renderKey")
		domain := r.URL.Query().Get("domain")

		contents, err := browser.RenderCSV(ctx, url, renderKey, domain)
		if err != nil {
			http.Error(w, "CSV rendering failed", http.StatusInternalServerError)
			slog.ErrorContext(ctx, "failed to render CSV", "err", err)
			return
		}

		w.Header().Set("Content-Type", "text/csv")
		w.Write(contents)
	})
}
