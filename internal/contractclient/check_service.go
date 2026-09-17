package contractclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"job4j.ru/share-trip/internal/service"
)

func (c *Client) CheckService(ctx context.Context, companyID string, serviceCode string) (service.CheckResult, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	resp, err := c.http.R().
		SetContext(ctx).
		SetBody(map[string]string{"client_id": companyID, "service_code": serviceCode}).
		Post("/contracts/check-service")
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
	return service.CheckResult{
		Allowed: *response.Allowed,
		Reason:  *response.Reason,
	}, nil
}
