package contractclient

import (
	"errors"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/go-resty/resty/v2"
)

var (
	ErrUnavailable     = errors.New("contract service unavailable")
	ErrInvalidResponse = errors.New("invalid contract service response")
)

type Client struct {
	http    *resty.Client
	timeout time.Duration
}

func New(baseURL string, timeout time.Duration, retryCount int) *Client {
	client := resty.New().
		SetBaseURL(baseURL).
		SetTimeout(timeout).
		SetRetryCount(max(0, retryCount)).
		SetRetryWaitTime(200 * time.Millisecond).
		SetRetryMaxWaitTime(1 * time.Second).
		SetRedirectPolicy(resty.RedirectPolicyFunc(func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		})).
		AddRetryCondition(func(r *resty.Response, err error) bool {
			if err != nil {
				var transportErr net.Error
				return errors.As(err, &transportErr) || errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF)
			}
			return r.StatusCode() == http.StatusTooManyRequests ||
				r.StatusCode() == http.StatusBadGateway ||
				r.StatusCode() == http.StatusServiceUnavailable ||
				r.StatusCode() == http.StatusGatewayTimeout
		})
	return &Client{http: client, timeout: timeout}
}
