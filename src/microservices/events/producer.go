package main

import (
	"context"

	"github.com/segmentio/kafka-go"
)

// Publisher abstracts writing a message to Kafka (allows mocking in tests).
type Publisher interface {
	Publish(ctx context.Context, topic, key string, value []byte) (partition int, offset int64, err error)
}

// KafkaProducer publishes messages to Kafka via a shared writer.
type KafkaProducer struct {
	writer *kafka.Writer
}

// NewKafkaProducer creates a producer connected to the given brokers.
func NewKafkaProducer(brokers []string) *KafkaProducer {
	return &KafkaProducer{
		writer: &kafka.Writer{
			Addr:     kafka.TCP(brokers...),
			Balancer: &kafka.LeastBytes{},
			// топик указывается per-message, поэтому здесь не фиксируем
		},
	}
}

// Publish writes a single message to the topic and returns its partition and offset.
func (p *KafkaProducer) Publish(ctx context.Context, topic, key string, value []byte) (int, int64, error) {
	msg := kafka.Message{
		Topic: topic,
		Key:   []byte(key),
		Value: value,
	}
	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		return 0, 0, err
	}
	// kafka-go не возвращает partition/offset из WriteMessages напрямую;
	// для MVP отдаём 0 — реальное смещение consumer логирует при чтении
	return 0, 0, nil
}

// Close flushes and closes the underlying writer.
func (p *KafkaProducer) Close() error {
	return p.writer.Close()
}
