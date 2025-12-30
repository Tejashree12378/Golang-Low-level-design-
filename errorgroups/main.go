package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/sync/errgroup"
)

const (
	timeout    = 1 * time.Second
	numWorkers = 3
)

func main() {
	jobs := make([]job, 10)
	for i := 1; i <= 10; i++ {
		jobs[i-1] = job{i, "https://www.google.com/", 0}
	}

	//jobs[5].url = "fksnef.com"
	//jobs[4].url = "https://httpbin.org/delay/5"
	jobs[4].sleepTime = 1

	batchProcessor(jobs)
}

type job struct {
	id        int
	url       string
	sleepTime time.Duration
}

func batchProcessor(jobs []job) error {
	parentCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var eg, ctx = errgroup.WithContext(parentCtx)

	tasks := make(chan job)

	for range numWorkers {
		eg.Go(func() error {
			return worker(ctx, tasks)
		})
	}

	go func() {
		defer close(tasks)
		for i := range jobs {
			select {
			case <-ctx.Done():
				return
			case tasks <- jobs[i]:
			}
		}
	}()

	eg.Go(func() error {
		return ctx.Err()
	})

	if err := eg.Wait(); err != nil {
		fmt.Println("failed to process this batch", err.Error())
		return err
	}

	fmt.Println("Batch Processed Successfully")
	return nil
}

func worker(ctx context.Context, jobs <-chan job) error {
	for j := range jobs {
		errCh := make(chan error, 1)

		go func(j job) {
			errCh <- executor(ctx, j)
		}(j)

		select {
		case err := <-errCh:
			if err != nil {
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return nil
}

func executor(ctx context.Context, j job) error {
	time.Sleep(j.sleepTime * time.Second)

	req, err := http.NewRequestWithContext(ctx, "GET", j.url, nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	fmt.Println(j.id, resp.StatusCode)

	return nil
}
