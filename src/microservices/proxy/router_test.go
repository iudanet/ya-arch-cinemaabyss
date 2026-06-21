package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testConfig returns a config pointing at fixed backend URLs for routing tests.
func testConfig() *Config {
	return &Config{
		MonolithURL:      "http://monolith",
		MoviesServiceURL: "http://movies",
		EventsServiceURL: "http://events",
	}
}

func TestChooseMoviesTarget(t *testing.T) {
	tests := []struct {
		name     string
		gradual  bool
		percent  int
		roll     int
		expected string
	}{
		{"migration off, percent 100 -> monolith", false, 100, 0, "http://monolith"},
		{"percent 0 -> monolith", true, 0, 0, "http://monolith"},
		{"percent 100 -> movies", true, 100, 99, "http://movies"},
		{"percent 50, roll 49 -> movies", true, 50, 49, "http://movies"},
		{"percent 50, roll 50 -> monolith (граница)", true, 50, 50, "http://monolith"},
		{"percent 50, roll 0 -> movies", true, 50, 0, "http://movies"},
		{"percent 50, roll 99 -> monolith", true, 50, 99, "http://monolith"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := testConfig()
			cfg.GradualMigration = tt.gradual
			cfg.MoviesMigrationPercent = tt.percent
			r := NewRouter(cfg, func() int { return tt.roll })

			got := r.chooseMoviesTarget(tt.roll)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestRouteTarget(t *testing.T) {
	cfg := testConfig()
	cfg.GradualMigration = true
	cfg.MoviesMigrationPercent = 100 // movies всегда уходит в микросервис
	r := NewRouter(cfg, func() int { return 0 })

	tests := []struct {
		path     string
		expected string
	}{
		{"/api/movies", "http://movies"},
		{"/api/movies/health", "http://movies"},
		{"/api/events/movie", "http://events"},
		{"/api/events/health", "http://events"},
		{"/api/users", "http://monolith"},
		{"/api/payments", "http://monolith"},
		{"/api/subscriptions", "http://monolith"},
		{"/anything-else", "http://monolith"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			assert.Equal(t, tt.expected, r.routeTarget(tt.path, 0))
		})
	}
}

// TestServeHTTPProxiesToBackend проверяет, что прокси прозрачно пробрасывает
// запрос на выбранный бэкенд и возвращает его тело и статус.
func TestServeHTTPProxiesToBackend(t *testing.T) {
	// поднимаем фейковый монолит
	monolith := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		assert.Equal(t, "/api/users", req.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[{"id":1}]`))
	}))
	defer monolith.Close()

	cfg := testConfig()
	cfg.MonolithURL = monolith.URL
	r := NewRouter(cfg, func() int { return 0 })

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	r.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)

	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.JSONEq(t, `[{"id":1}]`, string(body))
}

// TestServeHTTPMoviesGoesToMoviesService проверяет, что при percent=100
// movies-трафик уходит именно в movies-сервис.
func TestServeHTTPMoviesGoesToMoviesService(t *testing.T) {
	hit := ""
	movies := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hit = "movies"
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`))
	}))
	defer movies.Close()
	monolith := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hit = "monolith"
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`))
	}))
	defer monolith.Close()

	cfg := testConfig()
	cfg.MonolithURL = monolith.URL
	cfg.MoviesServiceURL = movies.URL
	cfg.GradualMigration = true
	cfg.MoviesMigrationPercent = 100
	r := NewRouter(cfg, func() int { return 0 })

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/movies", nil)
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Result().StatusCode)
	assert.Equal(t, "movies", hit)
}

func TestHealthHandler(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	healthHandler(rec, req)

	res := rec.Result()
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)

	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, "Strangler Fig Proxy is healthy", string(body))
}

func TestHealthHandlerRejectsPost(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/health", nil)
	healthHandler(rec, req)

	assert.Equal(t, http.StatusMethodNotAllowed, rec.Result().StatusCode)
}
