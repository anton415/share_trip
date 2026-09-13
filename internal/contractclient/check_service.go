package contractclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type CheckResult struct {
	Allowed bool
	Reason  string
}

func (c *Client) CheckService(ctx context.Context, companyID string, serviceCode string) (CheckResult, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	resp, err := c.http.R().
		SetContext(ctx).
		SetBody(map[string]string{"client_id": companyID, "service_code": serviceCode}).
		Post("/contracts/check-service")
	if err != nil {
		return CheckResult{}, fmt.Errorf("%w: %w", ErrUnavailable, err)
	}
	if resp.StatusCode() == http.StatusTooManyRequests || resp.StatusCode() >= http.StatusInternalServerError {
		return CheckResult{}, fmt.Errorf("%w: HTTP %d", ErrUnavailable, resp.StatusCode())
	}
	if resp.StatusCode() != http.StatusOK {
		return CheckResult{}, fmt.Errorf("%w: HTTP %d", ErrInvalidResponse, resp.StatusCode())
	}
	var response struct {
		Allowed *bool   `json:"allowed"`
		Reason  *string `json:"reason"`
	}
	if err := json.Unmarshal(resp.Body(), &response); err != nil || response.Allowed == nil || response.Reason == nil {
		return CheckResult{}, ErrInvalidResponse
	}
	return CheckResult{
		Allowed: *response.Allowed,
		Reason:  *response.Reason,
	}, nil
}
