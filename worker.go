package main

import (
	"context"
	"go-cloud-learning/monitor"
	"sync"
)

func worker(ctx context.Context, m *monitor.Monitor, jobs <-chan string, results chan<- monitor.Result, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			return

		case url, ok := <-jobs:
			if !ok {
				return
			}

			result := m.Check(ctx, url)

			select {
			case results <- result:

			case <-ctx.Done():
				return
			}
		}
	}

}
