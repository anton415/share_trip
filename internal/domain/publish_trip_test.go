package domain

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestNewEventID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		tripID    string
		eventType string
		want      string
	}{
		{
			name:      "same trip and event type keep the key",
			tripID:    "11111111-1111-4111-8111-111111111111",
			eventType: "TripPublished",
			want:      "59fb3430-6e67-5f95-8b93-c51508f081b3",
		},
		{
			name:      "another trip has another key",
			tripID:    "22222222-2222-4222-8222-222222222222",
			eventType: "TripPublished",
			want:      "bbbb58bd-d96d-5658-9a4d-52bbd5fb1dd8",
		},
		{
			name:      "another event type has another key",
			tripID:    "11111111-1111-4111-8111-111111111111",
			eventType: "TripCancelled",
			want:      "f02f544a-156c-5aeb-9976-664192f7137b",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tripID := uuid.MustParse(tt.tripID)
			want := uuid.MustParse(tt.want)
			eventID := newEventID(tripID, tt.eventType)
			require.Equal(t, want, eventID)
			require.Equal(t, eventID, newEventID(tripID, tt.eventType))
		})
	}
}
