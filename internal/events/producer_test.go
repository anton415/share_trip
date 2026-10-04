package events

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/require"

	"job4j.ru/share-trip/internal/domain"
)

func TestProducerPublishesMetadataHeaders(t *testing.T) {
	t.Parallel()
	brokers := os.Getenv("KAFKA_BROKERS")
	if brokers == "" {
		t.Skip("KAFKA_BROKERS is not set")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	topic := "sharetrip-metadata-test-" + uuid.NewString()
	client := &kafka.Client{Addr: kafka.TCP(strings.Split(brokers, ",")...)}
	created, err := client.CreateTopics(ctx, &kafka.CreateTopicsRequest{
		Topics: []kafka.TopicConfig{{Topic: topic, NumPartitions: 1, ReplicationFactor: 1}},
	})
	require.NoError(t, err)
	require.NoError(t, created.Errors[topic])
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		deleted, err := client.DeleteTopics(ctx, &kafka.DeleteTopicsRequest{Topics: []string{topic}})
		require.NoError(t, err)
		require.NoError(t, deleted.Errors[topic])
	})
	producer := NewProducer(strings.Split(brokers, ","), topic)
	t.Cleanup(func() { require.NoError(t, producer.Close()) })
	tripID := uuid.New()
	event := domain.TripPublished{
		EventID: uuid.NewSHA1(tripID, []byte("TripPublished")).String(), EventType: "TripPublished",
		CorrelationID: "pub-777", CausationID: "req-123",
		TraceParent: "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
		TripID:      tripID.String(), DriverID: uuid.NewString(), CompanyID: uuid.NewString(), OccurredAt: time.Now().UTC(),
	}
	require.NoError(t, producer.PublishTripPublished(ctx, event))
	reader := kafka.NewReader(kafka.ReaderConfig{Brokers: strings.Split(brokers, ","), Topic: topic})
	t.Cleanup(func() { require.NoError(t, reader.Close()) })
	message, err := reader.ReadMessage(ctx)
	require.NoError(t, err)
	require.Equal(t, event.TripID, string(message.Key))
	headers := make(map[string]string)
	for _, header := range message.Headers {
		headers[header.Key] = string(header.Value)
	}
	require.Equal(t, map[string]string{
		"event_id": event.EventID, "event_type": "TripPublished",
		"correlation_id": "pub-777", "causation_id": "req-123", "traceparent": event.TraceParent,
	}, headers)
	var payload domain.TripPublished
	require.NoError(t, json.Unmarshal(message.Value, &payload))
	require.Equal(t, event, payload)
}
