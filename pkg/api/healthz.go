package api

import "net/http"

func HandleGetHealthz() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tracer := tracer(r.Context())
		_, span := tracer.Start(r.Context(), "HandleGetHealthz")
		defer span.End()

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})
}
