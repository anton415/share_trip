package service

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

var ErrValidation = errors.New("validation error")

func validateCreateTripCommand(command CreateTripCommand) error {
	if command.DriverID == uuid.Nil {
		return errors.Join(ErrValidation, errors.New("driver_id is required"))
	}

	if strings.TrimSpace(command.FromPoint) == "" {
		return errors.Join(ErrValidation, errors.New("from_point is required"))
	}

	if strings.TrimSpace(command.ToPoint) == "" {
		return errors.Join(ErrValidation, errors.New("to_point is required"))
	}

	if !command.DepartureTime.After(time.Now()) {
		return errors.Join(ErrValidation, errors.New("departure_time must be in the future"))
	}

	if command.Seats <= 0 {
		return errors.Join(ErrValidation, errors.New("seats must be greater than zero"))
	}

	return nil
}
