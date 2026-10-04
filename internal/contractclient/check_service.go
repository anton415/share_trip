package contractclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.opentelemetry.io/otel/propagation"

	"job4j.ru/share-trip/internal/observability/logctx"
	"job4j.ru/share-trip/internal/observability/metrics"
	"job4j.ru/share-trip/internal/service"
)

func (c *Client) CheckService(ctx context.Context, companyID string, serviceCode string) (service.CheckResult, error) {
	started := time.Now()
	result := metrics.ResultError
	defer func() {
		c.metrics.ContractRequestTotal.WithLabelValues(result).Inc()
		c.metrics.ContractRequestDuration.WithLabelValues(result).Observe(time.Since(started).Seconds())
	}()
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	request := c.http.R().
		SetContext(ctx).
		SetHeader("X-Request-ID", logctx.RequestID(ctx)).
		SetHeader("X-Correlation-ID", logctx.CorrelationID(ctx)).
		SetHeader("X-Trip-ID", logctx.TripID(ctx)).
		SetBody(map[string]string{"client_id": companyID, "service_code": serviceCode})
	propagation.TraceContext{}.Inject(ctx, propagation.HeaderCarrier(request.Header))
	resp, err := request.Post("/contracts/check-service")
	if err != nil {
		return service.CheckResult{}, fmt.Errorf("%w: %w", ErrUnavailable, err)
	}
	if resp.StatusCode() == http.StatusTooManyRequests || resp.StatusCode() >= http.StatusInternalServerError {
		return service.CheckResult{}, fmt.Errorf("%w: HTTP %d", ErrUnavailable, resp.StatusCode())
	}
	if resp.StatusCode() != http.StatusOK {
		return service.CheckResult{}, fmt.Errorf("%w: HTTP %d", ErrInvalidResponse, resp.StatusCode())
	}
	var response struct {
		Allowed *bool   `json:"allowed"`
		Reason  *string `json:"reason"`
	}
	if err := json.Unmarshal(resp.Body(), &response); err != nil || response.Allowed == nil || response.Reason == nil {
		return service.CheckResult{}, ErrInvalidResponse
	}
	result = metrics.ResultSuccess
	return service.CheckResult{
		Allowed: *response.Allowed,
		Reason:  *response.Reason,
	}, nil
}
