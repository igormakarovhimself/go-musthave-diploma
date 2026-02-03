package accrual

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func TestAccrualClient_GetOrder_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/orders/12345678903" {
			t.Errorf("Expected path /api/orders/12345678903, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"order":"12345678903","status":"PROCESSED","accrual":500}`))
	}))
	defer server.Close()

	client := NewAccrualClient(server.URL)
	resp, statusCode, err := client.GetOrder(context.Background(), "12345678903")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if statusCode != http.StatusOK {
		t.Errorf("Expected status code 200, got %d", statusCode)
	}

	if resp == nil {
		t.Fatal("Expected response, got nil")
	}

	if resp.Order != "12345678903" {
		t.Errorf("Expected order 12345678903, got %s", resp.Order)
	}

	if resp.Status != "PROCESSED" {
		t.Errorf("Expected status PROCESSED, got %s", resp.Status)
	}

	expectedAccrual := decimal.NewFromInt(500)
	if resp.Accrual == nil || !resp.Accrual.Equal(expectedAccrual) {
		t.Errorf("Expected accrual 500, got %v", resp.Accrual)
	}
}

func TestAccrualClient_GetOrder_NoContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewAccrualClient(server.URL)
	resp, statusCode, err := client.GetOrder(context.Background(), "12345678903")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if statusCode != http.StatusNoContent {
		t.Errorf("Expected status code 204, got %d", statusCode)
	}

	if resp != nil {
		t.Errorf("Expected nil response, got %v", resp)
	}
}

func TestAccrualClient_GetOrder_TooManyRequests(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "60")
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte("Too many requests"))
	}))
	defer server.Close()

	client := NewAccrualClient(server.URL)
	resp, statusCode, err := client.GetOrder(context.Background(), "12345678903")

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if statusCode != http.StatusTooManyRequests {
		t.Errorf("Expected status code 429, got %d", statusCode)
	}

	if resp != nil {
		t.Errorf("Expected nil response, got %v", resp)
	}
}

func TestAccrualClient_GetOrder_InternalServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewAccrualClient(server.URL)
	resp, statusCode, err := client.GetOrder(context.Background(), "12345678903")

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if statusCode != http.StatusInternalServerError {
		t.Errorf("Expected status code 500, got %d", statusCode)
	}

	if resp != nil {
		t.Errorf("Expected nil response, got %v", resp)
	}
}

func TestAccrualClient_GetOrder_WithoutAccrual(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"order":"12345678903","status":"PROCESSING"}`))
	}))
	defer server.Close()

	client := NewAccrualClient(server.URL)
	resp, statusCode, err := client.GetOrder(context.Background(), "12345678903")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if statusCode != http.StatusOK {
		t.Errorf("Expected status code 200, got %d", statusCode)
	}

	if resp == nil {
		t.Fatal("Expected response, got nil")
	}

	if resp.Accrual != nil {
		t.Errorf("Expected nil accrual, got %v", resp.Accrual)
	}
}

func TestGetRetryAfter(t *testing.T) {
	tests := []struct {
		name            string
		retryAfter      string
		defaultDuration time.Duration
		expected        time.Duration
	}{
		{
			name:            "valid retry-after",
			retryAfter:      "60",
			defaultDuration: 10 * time.Second,
			expected:        60 * time.Second,
		},
		{
			name:            "empty retry-after",
			retryAfter:      "",
			defaultDuration: 10 * time.Second,
			expected:        10 * time.Second,
		},
		{
			name:            "invalid retry-after",
			retryAfter:      "invalid",
			defaultDuration: 10 * time.Second,
			expected:        10 * time.Second,
		},
		{
			name:            "nil response",
			retryAfter:      "",
			defaultDuration: 5 * time.Second,
			expected:        5 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var resp *http.Response
			if tt.name != "nil response" {
				resp = &http.Response{
					Header: http.Header{},
				}
				if tt.retryAfter != "" {
					resp.Header.Set("Retry-After", tt.retryAfter)
				}
			}

			result := GetRetryAfter(resp, tt.defaultDuration)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}
