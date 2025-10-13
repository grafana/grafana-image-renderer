package middleware

import (
	"log/slog"
	"net/http"
	"sync"
	"time"
)

func RequestLogger(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		lw := &loggingResponseWriter{w: w}
		defer func() {
			duration := time.Since(start)
			slog.InfoContext(r.Context(), "request complete",
				"method", r.Method,
				"mux_pattern", r.Pattern,
				"uri", r.URL,
				"status", lw.statusCode,
				"status_text", http.StatusText(lw.statusCode),
				"duration", duration)
		}()
		h.ServeHTTP(lw, r)
	})
}

type loggingResponseWriter struct {
	w          http.ResponseWriter
	once       sync.Once
	statusCode int
}

var (
	_ http.ResponseWriter = (*loggingResponseWriter)(nil)
	_ http.Flusher        = (*loggingResponseWriter)(nil)
)

func (l *loggingResponseWriter) Header() http.Header {
	return l.w.Header()
}

func (l *loggingResponseWriter) WriteHeader(code int) {
	l.once.Do(func() {
		l.statusCode = code
	})
	l.w.WriteHeader(code)
}

func (l *loggingResponseWriter) Write(b []byte) (int, error) {
	l.once.Do(func() {
		l.statusCode = http.StatusOK
	})
	return l.w.Write(b)
}

func (l *loggingResponseWriter) Flush() {
	if flusher, ok := l.w.(http.Flusher); ok {
		flusher.Flush()
	}
}
