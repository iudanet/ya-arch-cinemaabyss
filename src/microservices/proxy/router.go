package main

import (
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

// Router routes incoming requests to the monolith or to the extracted
// microservices, implementing the Strangler Fig pattern for the movies domain.
type Router struct {
	cfg  *Config
	roll func() int // источник «броска монетки» 0..99 (вынесен для тестируемости)
}

// NewRouter creates a Router with the given config and roll function.
func NewRouter(cfg *Config, roll func() int) *Router {
	return &Router{cfg: cfg, roll: roll}
}

// chooseMoviesTarget decides where /api/movies traffic goes:
// to the movies microservice (within the migration percentage) or to the monolith.
func (r *Router) chooseMoviesTarget(roll int) string {
	// постепенный перенос трафика по фиче-флагу
	if r.cfg.GradualMigration && roll < r.cfg.MoviesMigrationPercent {
		return r.cfg.MoviesServiceURL
	}
	return r.cfg.MonolithURL
}

// routeTarget selects the backend URL for the given request path.
// roll is the "coin toss" value used only for the movies migration decision.
func (r *Router) routeTarget(path string, roll int) string {
	switch {
	case strings.HasPrefix(path, "/api/movies"):
		return r.chooseMoviesTarget(roll)
	case strings.HasPrefix(path, "/api/events"):
		return r.cfg.EventsServiceURL
	default:
		// users, payments, subscriptions и всё остальное пока в монолите
		return r.cfg.MonolithURL
	}
}

// ServeHTTP routes the request to the chosen backend via a reverse proxy.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	target := r.routeTarget(req.URL.Path, r.roll())

	backend, err := url.Parse(target)
	if err != nil {
		slog.Error("invalid backend url", "target", target, "error", err)
		http.Error(w, `{"error":"bad gateway"}`, http.StatusBadGateway)
		return
	}

	// детальный лог выбранного бэкенда (общий лог запроса — в LoggingMiddleware)
	slog.Debug("routing decision", "path", req.URL.Path, "target", target)

	proxy := httputil.NewSingleHostReverseProxy(backend)
	proxy.ErrorHandler = func(w http.ResponseWriter, _ *http.Request, e error) {
		// бэкенд недоступен — отдаём 502 со структурированным телом
		slog.Error("backend unreachable", "target", target, "error", e)
		http.Error(w, `{"error":"bad gateway"}`, http.StatusBadGateway)
	}
	proxy.ServeHTTP(w, req)
}
