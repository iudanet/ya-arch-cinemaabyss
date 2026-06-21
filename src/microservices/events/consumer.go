package main

import (
	"context"
	"errors"
	"log/slog"

	"github.com/segmentio/kafka-go"
)

// Consumer reads messages from a single Kafka topic and logs their processing.
type Consumer struct {
	reader *kafka.Reader
	topic  string
}

// NewConsumer creates a consumer for the given topic and consumer group.
func NewConsumer(brokers []string, topic, groupID string) *Consumer {
	return &Consumer{
		topic: topic,
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers: brokers,
			Topic:   topic,
			GroupID: groupID,
		}),
	}
}

// Run consumes messages until the context is cancelled, logging each one.
func (c *Consumer) Run(ctx context.Context) {
	slog.Info("consumer started", "topic", c.topic)
	for {
		msg, err := c.reader.ReadMessage(ctx)
		if err != nil {
			// штатное завершение при отмене контекста
			if errors.Is(err, context.Canceled) {
				slog.Info("consumer stopped", "topic", c.topic)
				return
			}
			slog.Error("failed to read message", "topic", c.topic, "error", err)
			continue
		}

		// обработка события: для MVP пишем факт обработки в лог сервиса
		slog.Info("event processed",
			"topic", msg.Topic,
			"partition", msg.Partition,
			"offset", msg.Offset,
			"key", string(msg.Key),
			"value", string(msg.Value),
		)
	}
}

// Close closes the underlying reader.
func (c *Consumer) Close() error {
	return c.reader.Close()
}
