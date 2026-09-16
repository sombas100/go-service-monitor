package main

import (
	"context"
	"fmt"
	"go-cloud-learning/monitor"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
)

func main() {
	urls := []string{
		"https://example.com",
		"https://google.com",
		"https://github.com",
		"https://xuris.io",
	}
	config, err := loadConfig()
	if err != nil {
		slog.Error(
			"failed to load configuration",
			"error", err,
		)
		os.Exit(1)
	}

	m := &monitor.Monitor{
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
	results := make(chan monitor.Result)
	var wg sync.WaitGroup

	wg.Add(workerCount)

	for i := 0; i < workerCount; i++ {
		go worker(ctx, m, jobs, results, &wg)
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
