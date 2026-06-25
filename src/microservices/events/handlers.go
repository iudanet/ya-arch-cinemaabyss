package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// Handlers holds dependencies for the HTTP event handlers.
type Handlers struct {
	producer Publisher
	now      func() time.Time // вынесено для детерминированных тестов
}

// NewHandlers creates handlers with the given publisher.
func NewHandlers(producer Publisher) *Handlers {
	return &Handlers{
		producer: producer,
		now:      time.Now,
	}
}

// Register wires the handlers onto the mux using method-based routing (Go 1.22+).
func (h *Handlers) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/events/health", h.health)
	mux.HandleFunc("POST /api/events/movie", h.movieEvent)
	mux.HandleFunc("POST /api/events/user", h.userEvent)
	mux.HandleFunc("POST /api/events/payment", h.paymentEvent)
}

// health reports the events service health status.
func (h *Handlers) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{"status": true})
}

// movieEvent decodes a movie event and publishes it to the movie-events topic.
func (h *Handlers) movieEvent(w http.ResponseWriter, r *http.Request) {
	var in MovieEvent
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if in.MovieID == 0 || in.Title == "" || in.Action == "" {
		writeError(w, http.StatusBadRequest, "movie_id, title and action are required")
		return
	}
	h.publish(w, r.Context(), TopicMovieEvents, "movie", fmt.Sprintf("movie-%d", in.MovieID), in)
}

// userEvent decodes a user event and publishes it to the user-events topic.
func (h *Handlers) userEvent(w http.ResponseWriter, r *http.Request) {
	var in UserEvent
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if in.UserID == 0 || in.Action == "" {
		writeError(w, http.StatusBadRequest, "user_id and action are required")
		return
	}
	h.publish(w, r.Context(), TopicUserEvents, "user", fmt.Sprintf("user-%d", in.UserID), in)
}

// paymentEvent decodes a payment event and publishes it to the payment-events topic.
func (h *Handlers) paymentEvent(w http.ResponseWriter, r *http.Request) {
	var in PaymentEvent
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if in.PaymentID == 0 || in.UserID == 0 || in.Status == "" {
		writeError(w, http.StatusBadRequest, "payment_id, user_id and status are required")
		return
	}
	h.publish(w, r.Context(), TopicPaymentEvents, "payment", fmt.Sprintf("payment-%d", in.PaymentID), in)
}

// publish wraps the payload into an Event, sends it to Kafka and writes the response.
func (h *Handlers) publish(w http.ResponseWriter, ctx context.Context, topic, eventType, key string, payload any) {
	event := Event{
		ID:        key,
		Type:      eventType,
		Timestamp: h.now().UTC(),
		Payload:   payload,
	}

	value, err := json.Marshal(event)
	if err != nil {
		slog.Error("failed to marshal event", "type", eventType, "error", err)
		writeError(w, http.StatusInternalServerError, "failed to encode event")
		return
	}

	partition, offset, err := h.producer.Publish(ctx, topic, key, value)
	if err != nil {
		slog.Error("failed to publish event", "topic", topic, "error", err)
		writeError(w, http.StatusInternalServerError, "failed to publish event")
		return
	}

	slog.Info("event published", "topic", topic, "type", eventType, "key", key)

	writeJSON(w, http.StatusCreated, EventResponse{
		Status:    "success",
		Partition: partition,
		Offset:    offset,
		Event:     event,
	})
}

// writeJSON writes a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeError writes a standardized JSON error response.
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, ErrorResponse{Error: msg})
}
