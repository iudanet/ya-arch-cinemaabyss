package main

import (
	"log/slog"
	"math/rand"
	"net/http"
	"os"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg := LoadConfig()
	slog.Info("starting proxy service",
		"port", cfg.Port,
		"monolith", cfg.MonolithURL,
		"movies", cfg.MoviesServiceURL,
		"events", cfg.EventsServiceURL,
		"gradual_migration", cfg.GradualMigration,
		"movies_migration_percent", cfg.MoviesMigrationPercent,
	)

	// позапросный бросок монетки 0..99 для решения о переносе movies-трафика
	router := NewRouter(cfg, func() int { return rand.Intn(100) })

	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	mux.Handle("/", router)

	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: LoggingMiddleware(mux),
	}

	if err := srv.ListenAndServe(); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

// healthHandler reports the proxy health status.
func healthHandler(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("Strangler Fig Proxy is healthy"))
}
