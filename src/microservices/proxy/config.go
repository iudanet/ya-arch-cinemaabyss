package main

import (
	"log/slog"
	"os"
	"strconv"
)

// Config holds the proxy service configuration read from environment variables.
type Config struct {
	Port                   string
	MonolithURL            string
	MoviesServiceURL       string
	EventsServiceURL       string
	GradualMigration       bool
	MoviesMigrationPercent int
}

// LoadConfig reads the proxy configuration from environment variables,
// applying defaults that match docker-compose.
func LoadConfig() *Config {
	cfg := &Config{
		Port:                   getEnv("PORT", "8000"),
		MonolithURL:            getEnv("MONOLITH_URL", "http://localhost:8080"),
		MoviesServiceURL:       getEnv("MOVIES_SERVICE_URL", "http://localhost:8081"),
		EventsServiceURL:       getEnv("EVENTS_SERVICE_URL", "http://localhost:8082"),
		GradualMigration:       getEnvBool("GRADUAL_MIGRATION", false),
		MoviesMigrationPercent: getEnvInt("MOVIES_MIGRATION_PERCENT", 0),
	}

	// процент миграции ограничиваем диапазоном 0..100
	if cfg.MoviesMigrationPercent < 0 {
		cfg.MoviesMigrationPercent = 0
	}
	if cfg.MoviesMigrationPercent > 100 {
		cfg.MoviesMigrationPercent = 100
	}

	return cfg
}

// getEnv returns the value of the environment variable or a default.
func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// getEnvBool parses a boolean environment variable, falling back to a default.
func getEnvBool(key string, def bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		slog.Warn("invalid bool env, using default", "key", key, "value", v, "default", def)
		return def
	}
	return b
}

// getEnvInt parses an integer environment variable, falling back to a default.
func getEnvInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		slog.Warn("invalid int env, using default", "key", key, "value", v, "default", def)
		return def
	}
	return n
}
