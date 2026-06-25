package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockPublisher is a hand-written mock of Publisher (single method).
type mockPublisher struct {
	calledTopic string
	calledKey   string
	calledValue []byte
	retErr      error
}

func (m *mockPublisher) Publish(_ context.Context, topic, key string, value []byte) (int, int64, error) {
	m.calledTopic = topic
	m.calledKey = key
	m.calledValue = value
	if m.retErr != nil {
		return 0, 0, m.retErr
	}
	return 3, 42, nil
}

// newTestHandlers builds handlers with a fixed clock for deterministic output.
func newTestHandlers(p Publisher) *Handlers {
	h := NewHandlers(p)
	h.now = func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) }
	return h
}

func doRequest(h *Handlers, method, path, body string) *httptest.ResponseRecorder {
	mux := http.NewServeMux()
	h.Register(mux)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	mux.ServeHTTP(rec, req)
	return rec
}

func TestHealth(t *testing.T) {
	h := newTestHandlers(&mockPublisher{})
	rec := doRequest(h, http.MethodGet, "/api/events/health", "")

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `{"status":true}`, rec.Body.String())
}

func TestMovieEventSuccess(t *testing.T) {
	mp := &mockPublisher{}
	h := newTestHandlers(mp)
	body := `{"movie_id":1,"title":"Inception","action":"viewed","user_id":7}`

	rec := doRequest(h, http.MethodPost, "/api/events/movie", body)

	require.Equal(t, http.StatusCreated, rec.Code)
	assert.Contains(t, rec.Body.String(), `"status":"success"`)
	assert.Contains(t, rec.Body.String(), `"partition":3`)
	assert.Contains(t, rec.Body.String(), `"offset":42`)
	// проверяем, что улетело в нужный топик с нужным ключом
	assert.Equal(t, TopicMovieEvents, mp.calledTopic)
	assert.Equal(t, "movie-1", mp.calledKey)
}

func TestUserEventSuccess(t *testing.T) {
	mp := &mockPublisher{}
	h := newTestHandlers(mp)
	body := `{"user_id":7,"username":"john","action":"logged_in","timestamp":"2026-01-01T00:00:00Z"}`

	rec := doRequest(h, http.MethodPost, "/api/events/user", body)

	require.Equal(t, http.StatusCreated, rec.Code)
	assert.Equal(t, TopicUserEvents, mp.calledTopic)
	assert.Equal(t, "user-7", mp.calledKey)
}

func TestPaymentEventSuccess(t *testing.T) {
	mp := &mockPublisher{}
	h := newTestHandlers(mp)
	body := `{"payment_id":5,"user_id":7,"amount":9.99,"status":"completed","timestamp":"2026-01-01T00:00:00Z"}`

	rec := doRequest(h, http.MethodPost, "/api/events/payment", body)

	require.Equal(t, http.StatusCreated, rec.Code)
	assert.Equal(t, TopicPaymentEvents, mp.calledTopic)
	assert.Equal(t, "payment-5", mp.calledKey)
}

func TestMovieEventValidation(t *testing.T) {
	h := newTestHandlers(&mockPublisher{})
	// нет обязательных полей title/action
	rec := doRequest(h, http.MethodPost, "/api/events/movie", `{"movie_id":1}`)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestMovieEventInvalidJSON(t *testing.T) {
	h := newTestHandlers(&mockPublisher{})
	rec := doRequest(h, http.MethodPost, "/api/events/movie", `{not json`)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestPublishErrorReturns500(t *testing.T) {
	mp := &mockPublisher{retErr: errors.New("kafka down")}
	h := newTestHandlers(mp)
	body := `{"movie_id":1,"title":"Inception","action":"viewed"}`

	rec := doRequest(h, http.MethodPost, "/api/events/movie", body)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestWrongMethodReturns405(t *testing.T) {
	h := newTestHandlers(&mockPublisher{})
	// GET на POST-эндпоинт — ServeMux вернёт 405
	rec := doRequest(h, http.MethodGet, "/api/events/movie", "")
	assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
}
