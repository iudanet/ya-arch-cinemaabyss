package main

import (
	"log/slog"
	"net/http"
	"time"
)

// statusRecorder wraps http.ResponseWriter to capture the response status code.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

// WriteHeader records the status code before delegating to the wrapped writer.
func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// LoggingMiddleware logs each incoming request: method, path, status and duration.
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		start := time.Now()
		// статус по умолчанию 200 — если обработчик не вызовет WriteHeader явно
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(rec, req)

		slog.Info("request handled",
			"method", req.Method,
			"path", req.URL.Path,
			"status", rec.status,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}
