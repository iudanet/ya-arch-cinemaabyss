package main

import "time"

// Topic names match KAFKA_CREATE_TOPICS in docker-compose.
const (
	TopicMovieEvents   = "movie-events"
	TopicUserEvents    = "user-events"
	TopicPaymentEvents = "payment-events"
)

// MovieEvent is the input payload for a movie-related event.
type MovieEvent struct {
	MovieID     int      `json:"movie_id"`
	Title       string   `json:"title"`
	Action      string   `json:"action"`
	UserID      int      `json:"user_id,omitempty"`
	Rating      float64  `json:"rating,omitempty"`
	Genres      []string `json:"genres,omitempty"`
	Description string   `json:"description,omitempty"`
}

// UserEvent is the input payload for a user-related event.
type UserEvent struct {
	UserID    int       `json:"user_id"`
	Username  string    `json:"username,omitempty"`
	Email     string    `json:"email,omitempty"`
	Action    string    `json:"action"`
	Timestamp time.Time `json:"timestamp"`
}

// PaymentEvent is the input payload for a payment-related event.
type PaymentEvent struct {
	PaymentID  int       `json:"payment_id"`
	UserID     int       `json:"user_id"`
	Amount     float64   `json:"amount"`
	Status     string    `json:"status"`
	Timestamp  time.Time `json:"timestamp"`
	MethodType string    `json:"method_type,omitempty"`
}

// Event is the envelope written to Kafka and returned to the caller.
type Event struct {
	ID        string      `json:"id"`
	Type      string      `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
	Payload   any       `json:"payload"`
}

// EventResponse is the HTTP response after publishing an event to Kafka.
type EventResponse struct {
	Status    string `json:"status"`
	Partition int    `json:"partition"`
	Offset    int64  `json:"offset"`
	Event     Event  `json:"event"`
}

// ErrorResponse is the standardized error body.
type ErrorResponse struct {
	Error string `json:"error"`
}
