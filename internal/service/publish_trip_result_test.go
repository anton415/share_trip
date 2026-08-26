package service

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"job4j.ru/share-trip/internal/domain"
	observability "job4j.ru/share-trip/internal/observability/metrics"
)

func TestPublishTripResult(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "not found",
			err:  domain.ErrTripNotFound,
			want: observability.ResultNotFound,
		},
		{
			name: "forbidden",
			err:  domain.ErrForbidden,
			want: observability.ResultForbidden,
		},
		{
			name: "conflict",
			err:  domain.ErrConflict,
			want: observability.ResultConflict,
		},
		{
			name: "already published",
			err:  domain.ErrTripAlreadyPublished,
			want: observability.ResultAlreadyPublished,
		},
		{
			name: "wrapped conflict",
			err:  fmt.Errorf("transaction: %w", domain.ErrConflict),
			want: observability.ResultConflict,
		},
		{
			name: "internal error",
			err:  errors.New("database unavailable"),
			want: observability.ResultInternalError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, publishTripResult(tt.err))
		})
	}
}
