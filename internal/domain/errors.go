package domain

import "errors"

var (
	ErrTripNotFound         = errors.New("trip not found")
	ErrForbidden            = errors.New("forbidden")
	ErrConflict             = errors.New("conflict")
	ErrTripAlreadyPublished = errors.New("trip already published")
)
