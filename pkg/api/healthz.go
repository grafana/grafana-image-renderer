package api

import (
	"log/slog"
	"net/http"

	"go.opentelemetry.io/otel/codes"
)

func HandleGetHealthz() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tracer := tracer(r.Context())
		ctx, span := tracer.Start(r.Context(), "HandleGetHealthz")
		defer span.End()

		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("OK")); err != nil {
			slog.ErrorContext(ctx, "failed to write health response", "error", err)
			span.SetStatus(codes.Error, "failed to write health response")
			span.RecordError(err)
		}
	})
}
