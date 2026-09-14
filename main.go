package main

import (
	// "cloudmonitor/resource"
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
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
}

type Result struct {
	URL        string
	StatusCode int
	Latency    time.Duration
	Healthy    bool
	Err        error
}

func main() {
	urls := []string{
		"https://example.com",
		"https://google.com",
		"https://github.com",
	}

	monitor := &Monitor{
		Client:           &http.Client{},
		Timeout:          2 * time.Second,
		HealthyThreshold: 1 * time.Second,
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

	for _, url := range urls {
		jobs <- url
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
