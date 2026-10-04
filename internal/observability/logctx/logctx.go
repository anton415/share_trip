package logctx

import (
	"context"
	"log/slog"
)

type loggerKey struct{}
type requestIDKey struct{}
type correlationIDKey struct{}
type tripIDKey struct{}

func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, logger)
}

func Logger(ctx context.Context) *slog.Logger {
	logger, ok := ctx.Value(loggerKey{}).(*slog.Logger)
	if !ok || logger == nil {
		return slog.Default()
	}
	return logger
}

func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, requestID)
}

func RequestID(ctx context.Context) string {
	value, ok := ctx.Value(requestIDKey{}).(string)
	if !ok {
		return ""
	}
	return value
}

func WithCorrelationID(ctx context.Context, correlationID string) context.Context {
	return context.WithValue(ctx, correlationIDKey{}, correlationID)
}

func CorrelationID(ctx context.Context) string {
	value, ok := ctx.Value(correlationIDKey{}).(string)
	if !ok {
		return ""
	}
	return value
}

func WithTripID(ctx context.Context, tripID string) context.Context {
	return context.WithValue(ctx, tripIDKey{}, tripID)
}

func TripID(ctx context.Context) string {
	value, ok := ctx.Value(tripIDKey{}).(string)
	if !ok {
		return ""
	}
	return value
}
