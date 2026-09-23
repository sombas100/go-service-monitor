package monitor

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestMonitorHealthyService(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)
	defer server.Close()

	m := &Monitor{
		Client:           server.Client(),
		Timeout:          1 * time.Second,
		HealthyThreshold: 500 * time.Millisecond,
		MaxRetries:       0,
		RetryDelay:       10 * time.Millisecond,
	}

	result := m.Check(context.Background(), server.URL)

	if !result.Healthy {
		t.Error("Service expected to be healthy")
	}

	if result.StatusCode != http.StatusOK {
		t.Errorf("Expected status code 200, received %d", result.StatusCode)
	}

	if result.Err != nil {
		t.Errorf("Expected no error, received %v", result.Err)
	}
}

func TestMonitorUnhealthyService(t *testing.T) {

	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}),
	)

	defer server.Close()

	m := &Monitor{
		Client:           server.Client(),
		Timeout:          1 * time.Second,
		HealthyThreshold: 500 * time.Millisecond,
		MaxRetries:       0,
		RetryDelay:       10 * time.Millisecond,
	}

	result := m.Check(context.Background(), server.URL)

	if result.Healthy {
		t.Error("Service expected to be unhealthy")
	}

	if result.StatusCode != http.StatusInternalServerError {
		t.Errorf("Expected status 500, received %d", result.StatusCode)
	}

	if result.Err != nil {
		t.Errorf("Expected an error, received %v", result.Err)
	}
}

func TestMonitorRetriesServerError(t *testing.T) {
	var requestCount atomic.Int32
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestCount.Add(1)
			w.WriteHeader(http.StatusInternalServerError)
		}),
	)

	defer server.Close()

	m := &Monitor{
		Client:           server.Client(),
		Timeout:          1 * time.Second,
		HealthyThreshold: 500 * time.Millisecond,
		MaxRetries:       2,
		RetryDelay:       10 * time.Millisecond,
	}

	m.Check(context.Background(), server.URL)

	count := requestCount.Load()

	if count != 3 {
		t.Errorf("Expected three attempts, received %d", count)
	}
}

func TestMonitorTimeout(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			select {
			case <-time.After(500 * time.Millisecond):
				w.WriteHeader(http.StatusOK)
			case <-r.Context().Done():
				return
			}
		}),
	)

	defer server.Close()

	m := &Monitor{
		Client:           server.Client(),
		Timeout:          50 * time.Millisecond,
		HealthyThreshold: 500 * time.Millisecond,
		MaxRetries:       0,
		RetryDelay:       10 * time.Millisecond,
	}

	result := m.Check(context.Background(), server.URL)

	if result.Healthy {
		t.Error("Expected timed-out service to be unhealthy")
	}

	if result.StatusCode != 0 {
		t.Errorf("Expected a status code 0, received: %d", result.StatusCode)
	}

	if !errors.Is(result.Err, context.DeadlineExceeded) {
		t.Errorf("Expected deadline error exceeded, received: %v", result.Err)
	}
}

func TestMonitorStopsRetryingAfterRecovery(t *testing.T) {
	var requestCount atomic.Int32

	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			count := requestCount.Add(1)

			if count <= 2 {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusOK)
		}),
	)

	defer server.Close()

	m := &Monitor{
		Client:           server.Client(),
		Timeout:          2 * time.Second,
		HealthyThreshold: 500 * time.Millisecond,
		MaxRetries:       5,
		RetryDelay:       10 * time.Millisecond,
	}

	result := m.Check(context.Background(), server.URL)
	count := requestCount.Load()

	if count != 3 {
		t.Errorf("Expected 3 requests, received: %d", count)
	}

	if !result.Healthy {
		t.Errorf("Expected server to be healthy, received: %t", result.Healthy)
	}

	if result.StatusCode != http.StatusOK {
		t.Errorf("Expected status of 200, received: %d", result.StatusCode)
	}

	if result.Err != nil {
		t.Errorf("Expected no errors, received: %v", result.Err)
	}
}
