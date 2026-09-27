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
	})
}

func (p *Producer) Close() error {
	return p.writer.Close()
}
