package service

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestValidateCreateTripCommand(t *testing.T) {
	t.Parallel()

	command := CreateTripCommand{
		DriverID:      uuid.Nil,
		FromPoint:     "Moscow",
		ToPoint:       "Saint Petersburg",
		DepartureTime: time.Now().Add(time.Hour),
		Seats:         1,
	}

	err := validateCreateTripCommand(command)

	require.ErrorIs(t, err, ErrValidation)
	require.Contains(t, err.Error(), "driver_id is required")
}
