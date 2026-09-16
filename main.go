package main

import (
	// "cloudmonitor/resource"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"time"
)

// type Tracker interface {
// 	IsOnLeave() bool
// 	ResourceName() string
// }

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

type Config struct {
	Timeout          time.Duration
	HealthyThreshold time.Duration
	MaxRetries       int
	RetryDelay       time.Duration
}

func main() {
	urls := []string{
		"https://example.com",
		"https://google.com",
		"https://github.com",
		"https://interlu.iox",
	}
	config, err := loadConfig()
	if err != nil {
		slog.Error(
			"failed to load configuration",
			"error", err,
		)
		os.Exit(1)
	}

	monitor := &Monitor{
		Client:           &http.Client{},
		Timeout:          config.Timeout,
		HealthyThreshold: config.HealthyThreshold,
		MaxRetries:       config.MaxRetries,
		RetryDelay:       config.RetryDelay,
	}

	workerCount := 3
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
	)
	defer stop()

	jobs := make(chan string, len(urls))
	results := make(chan Result)
	var wg sync.WaitGroup

	wg.Add(workerCount)

	for i := 0; i < workerCount; i++ {
		go worker(ctx, monitor, jobs, results, &wg)
	}

enqueueJobs:
	for _, url := range urls {

		select {
		case <-ctx.Done():
			break enqueueJobs

		case jobs <- url:
		}

	}

	close(jobs)

	go func() {
		wg.Wait()
		close(results)
	}()

	for result := range results {
		fmt.Println("---------------------------------------")
		fmt.Println("URL:", result.URL)
		fmt.Println("Latency:", result.Latency)
		fmt.Println("Healthy:", result.Healthy)

		if result.Err != nil {
			fmt.Println("Error:", result.Err)
		} else {
			fmt.Println("Status:", result.StatusCode)
		}
	}
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

func worker(ctx context.Context, monitor *Monitor, jobs <-chan string, results chan<- Result, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			return

		case url, ok := <-jobs:
			if !ok {
				return
			}

			result := monitor.Check(ctx, url)

			select {
			case results <- result:
				// successfully sent result

			case <-ctx.Done():
				return
			}
		}
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

func getDurationEnv(key string, defaultValue time.Duration) (time.Duration, error) {
	value, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue, nil
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", key, err)
	}

	return duration, nil
}

func getIntEnv(key string, defaultValue int) (int, error) {
	value, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue, nil
	}
	parsedValue, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", key, err)
	}

	return parsedValue, nil
}

func loadConfig() (Config, error) {
	timeout, err := getDurationEnv("MONITOR_TIMEOUT", 2*time.Second)
	if err != nil {
		return Config{}, err
	}

	health, err := getDurationEnv("MONITOR_HEALTHY_THRESHOLD", 1*time.Second)
	if err != nil {
		return Config{}, err
	}

	retries, err := getIntEnv("MONITOR_MAX_RETRIES", 3)
	if err != nil {
		return Config{}, err
	}

	delay, err := getDurationEnv("MONITOR_RETRY_DELAY", 500*time.Millisecond)
	if err != nil {
		return Config{}, err
	}

	config := Config{
		Timeout:          timeout,
		HealthyThreshold: health,
		MaxRetries:       retries,
		RetryDelay:       delay,
	}
	err = config.Validate()
	if err != nil {
		return Config{}, err
	}
	return config, nil
}

func (c Config) Validate() error {
	if c.MaxRetries < 0 {
		return fmt.Errorf("MONITOR_MAX_RETRIES cannot be negative: %d", c.MaxRetries)
	}

	if c.Timeout <= 0 {
		return fmt.Errorf("MONITOR_TIMEOUT cannot be 0 or less: %s", c.Timeout)
	}

	if c.HealthyThreshold <= 0 {
		return fmt.Errorf(
			"MONITOR_HEALTHY_THRESHOLD must be greater than 0: %s",
			c.HealthyThreshold,
		)
	}

	if c.RetryDelay <= 0 {
		return fmt.Errorf(
			"MONITOR_RETRY_DELAY must be greater than 0: %s",
			c.RetryDelay,
		)
	}

	return nil
}

// func inspectResource(resource Monitor) {
// 	fmt.Println("Initiating resource for:", resource.ResourceName())

// 	defer fmt.Println("Closing resource...")

// 	if !resource.IsHealthy() {
// 		fmt.Println("Resource is unhealthy")
// 		return
// 	}

// 	fmt.Println("Resource is healthy")
// }

// func checkResource(resource Monitor, results chan<- string) {
// 	if resource.IsHealthy() {
// 		results <- resource.ResourceName() + " is healthy"
// 		return
// 	}

// 	results <- resource.ResourceName() + " is unhealthy"
// }
