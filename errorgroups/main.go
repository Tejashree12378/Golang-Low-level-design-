package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/sync/errgroup"
)

const (
	timeout    = 500 * time.Millisecond
	numWorkers = 1
)

func main() {
	jobs := make([]job, 10)
	for i := 1; i <= 10; i++ {
		jobs[i-1] = job{i, "https://www.google.com/"}
	}

	//jobs[3].url = "fksnef.com"
	//jobs[4].url = "https://httpbin.org/delay/5"

	batchProcessor(jobs)
}

type job struct {
	id  int
	url string
}

func batchProcessor(jobs []job) error {
	parentCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var eg, ctx = errgroup.WithContext(parentCtx)
	eg.SetLimit(numWorkers)

	for _, j := range jobs {
		j := j

		if ctx.Err() != nil {
			break
		}

		eg.Go(func() error {
			if ctx.Err() != nil {
				return ctx.Err()
			}

			if j.id%3 == 0 {
				time.Sleep(1 * time.Second)
			}

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
		})
	}

	eg.Go(func() error {
		return ctx.Err()
	})

	if err := eg.Wait(); err != nil {
		fmt.Println("failed to process this batch", err.Error())
		return err
	}

	return nil
}
