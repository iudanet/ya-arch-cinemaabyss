package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	port := getEnv("PORT", "8082")
	brokers := strings.Split(getEnv("KAFKA_BROKERS", "localhost:9092"), ",")
	slog.Info("starting events service", "port", port, "brokers", brokers)

	// producer для записи событий
	producer := NewKafkaProducer(brokers)
	defer producer.Close()

	// consumer'ы на каждый топик читают и логируют обработку
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	topics := []string{TopicMovieEvents, TopicUserEvents, TopicPaymentEvents}
	consumers := make([]*Consumer, 0, len(topics))
	for _, topic := range topics {
		c := NewConsumer(brokers, topic, "events-service")
		consumers = append(consumers, c)
		go c.Run(ctx)
	}

	// HTTP-сервер с method-based роутингом
	handlers := NewHandlers(producer)
	mux := http.NewServeMux()
	handlers.Register(mux)

	srv := &http.Server{Addr: ":" + port, Handler: mux}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server stopped", "error", err)
			os.Exit(1)
		}
	}()

	// ждём сигнал завершения и аккуратно гасим
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	slog.Info("shutting down events service")
	cancel()
	for _, c := range consumers {
		_ = c.Close()
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	_ = srv.Shutdown(shutdownCtx)
}

// getEnv returns the value of the environment variable or a default.
func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
