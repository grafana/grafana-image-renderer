package api

import (
	"log/slog"
	"net/http"
)

func HandleGetHealthz() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tracer := tracer(r.Context())
		_, span := tracer.Start(r.Context(), "HandleGetHealthz")
		defer span.End()

		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("OK")); err != nil {
			slog.Error("failed to write health response", "error", err)
			span.RecordError(err)
		}
	})
}