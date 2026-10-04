package events

import (
	"context"
	"encoding/json"

	"github.com/segmentio/kafka-go"

	"job4j.ru/share-trip/internal/domain"
)

type Producer struct {
	writer *kafka.Writer
}

func NewProducer(brokers []string, topic string) *Producer {
	return &Producer{
		writer: &kafka.Writer{
			Addr:         kafka.TCP(brokers...),
			Topic:        topic,
			Balancer:     &kafka.Hash{},
			RequiredAcks: kafka.RequireAll,
		},
	}
}

func (p *Producer) PublishTripPublished(ctx context.Context, event domain.TripPublished) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(event.TripID),
		Value: data,
		Headers: []kafka.Header{
			{Key: "event_id", Value: []byte(event.EventID)},
			{Key: "event_type", Value: []byte(event.EventType)},
			{Key: "correlation_id", Value: []byte(event.CorrelationID)},
			{Key: "causation_id", Value: []byte(event.CausationID)},
			{Key: "traceparent", Value: []byte(event.TraceParent)},
		},
	})
}

func (p *Producer) Close() error {
	return p.writer.Close()
}
