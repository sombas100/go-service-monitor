package monitor

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"time"
)

type Monitor struct {
	Client           *http.Client
	Timeout          time.Duration
	HealthyThreshold time.Duration
	MaxRetries       int
	RetryDelay       time.Duration
}

type Result struct {
	URL        string
	StatusCode int
	Latency    time.Duration
	Healthy    bool
	Err        error
}

func (m *Monitor) Check(ctx context.Context, url string) Result {

	for attempt := 0; attempt <= m.MaxRetries; attempt++ {
		slog.Info("Checking service...", "url", url, "attempt", attempt)
		result := m.checkOnce(ctx, url)

		if ctx.Err() != nil {
			return result
		}

		if !shouldRetry(result) {
			return result
		}

		if attempt == m.MaxRetries {
			slog.Error(
				"Service check failed after retries",
				"url", url,
				"attempts", attempt+1,
				"status", result.StatusCode,
				"error", result.Err,
			)
			return result
		}

		delay := m.RetryDelay * time.Duration(1<<attempt)
		slog.Warn(
			"Service check failed, retrying...",
			"url", url,
			"attempt", attempt,
			"status", result.StatusCode,
			"error", result.Err,
			"retry_delay", delay,
		)

		select {
		case <-ctx.Done():
			return result
		case <-time.After(delay):
		}
	}
	return Result{}
}

func (m *Monitor) checkOnce(ctx context.Context, url string) Result {

	requestCtx, cancel := context.WithTimeout(
		ctx,
		m.Timeout,
	)
	defer cancel()

	req, err := http.NewRequestWithContext(
		requestCtx,
		http.MethodGet,
		url,
		nil,
	)

	if err != nil {
		return Result{
			URL:     url,
			Healthy: false,
			Err:     err,
		}
	}

	start := time.Now()
	resp, err := m.Client.Do(req)
	elapsed := time.Since(start)

	if err != nil {
		return Result{
			URL:     url,
			Latency: elapsed,
			Healthy: false,
			Err:     err,
		}
	}

	defer resp.Body.Close()

	healthy := resp.StatusCode >= 200 &&
		resp.StatusCode < 300 &&
		elapsed <= m.HealthyThreshold

	return Result{
		URL:        url,
		StatusCode: resp.StatusCode,
		Latency:    elapsed,
		Healthy:    healthy,
		Err:        nil,
	}
}

func shouldRetry(result Result) bool {
	var dnsErr *net.DNSError
	if errors.As(result.Err, &dnsErr) && dnsErr.IsNotFound {
		return false
	}
	if result.StatusCode >= 500 {
		return true
	}

	if errors.Is(result.Err, context.DeadlineExceeded) {
		return true
	}

	return false
}
