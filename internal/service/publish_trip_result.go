package service

import (
	"errors"

	"job4j.ru/share-trip/internal/domain"
	observability "job4j.ru/share-trip/internal/observability/metrics"
)

func publishTripResult(err error) string {
	switch {
	case errors.Is(err, domain.ErrTripNotFound):
		return observability.ResultNotFound
	case errors.Is(err, domain.ErrForbidden):
		return observability.ResultForbidden
	case errors.Is(err, domain.ErrConflict):
		return observability.ResultConflict
	case errors.Is(err, domain.ErrTripAlreadyPublished):
		return observability.ResultAlreadyPublished
	default:
		return observability.ResultInternalError
	}
}
