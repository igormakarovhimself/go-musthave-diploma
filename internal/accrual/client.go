package accrual

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/shopspring/decimal"
)

type AccrualClient struct {
	baseURL    string
	httpClient *http.Client
}

type AccrualResponse struct {
	Order   string           `json:"order"`
	Status  string           `json:"status"`
	Accrual *decimal.Decimal `json:"accrual,omitempty"`
}

func NewAccrualClient(baseURL string) *AccrualClient {
	return &AccrualClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *AccrualClient) GetOrder(ctx context.Context, number string) (*AccrualResponse, int, error) {
	url := fmt.Sprintf("%s/api/orders/%s", c.baseURL, number)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	statusCode := resp.StatusCode

	if statusCode == http.StatusNoContent {
		return nil, statusCode, nil
	}

	if statusCode == http.StatusTooManyRequests {
		return nil, statusCode, fmt.Errorf("too many requests")
	}

	if statusCode == http.StatusInternalServerError {
		return nil, statusCode, fmt.Errorf("internal server error")
	}

	if statusCode != http.StatusOK {
		return nil, statusCode, fmt.Errorf("unexpected status code: %d", statusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, statusCode, fmt.Errorf("failed to read response body: %w", err)
	}

	var accrualResp AccrualResponse
	if err := json.Unmarshal(body, &accrualResp); err != nil {
		return nil, statusCode, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &accrualResp, statusCode, nil
}

func GetRetryAfter(resp *http.Response, defaultDuration time.Duration) time.Duration {
	if resp == nil {
		return defaultDuration
	}

	retryAfter := resp.Header.Get("Retry-After")
	if retryAfter == "" {
		return defaultDuration
	}

	seconds, err := strconv.Atoi(retryAfter)
	if err != nil {
		return defaultDuration
	}

	return time.Duration(seconds) * time.Second
}
