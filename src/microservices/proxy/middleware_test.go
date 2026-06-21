package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestLoggingMiddlewarePassesThrough проверяет, что middleware не ломает
// ответ обёрнутого обработчика (статус и тело проходят насквозь).
func TestLoggingMiddlewarePassesThrough(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("ok"))
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/movies", nil)

	LoggingMiddleware(handler).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Result().StatusCode)
	assert.Equal(t, "ok", rec.Body.String())
}

// TestStatusRecorderDefaultsTo200 проверяет, что без явного WriteHeader
// статус по умолчанию остаётся 200.
func TestStatusRecorderDefaultsTo200(t *testing.T) {
	rec := &statusRecorder{ResponseWriter: httptest.NewRecorder(), status: http.StatusOK}
	_, _ = rec.Write([]byte("no explicit header"))
	assert.Equal(t, http.StatusOK, rec.status)
}
