package main

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

const (
	NUM_WORKERS = 5
	TIMEOUT     = 20 * time.Second
)

func main() {
	dispatcher()
}

func dispatcher() {
	signal := make(chan int)
	parentCtx, cancel := context.WithTimeout(context.Background(), TIMEOUT)
	defer cancel()

	var eg, ctx = errgroup.WithContext(parentCtx)

	for i := 0; i < NUM_WORKERS; i++ {
		eg.Go(func() error {
			return consumer(signal, ctx)
		})

		eg.Go(func() error {
			return producer(signal, ctx)
		})
	}

	if err := eg.Wait(); err != nil {
		if err.Error() == "context deadline exceeded" {
			fmt.Printf("processing stopped after %v seconds", TIMEOUT)
			return
		}

		fmt.Println(err)
	}
}

var (
	Queue []Event
	mu    sync.Mutex
	ID    int
)

type Event struct {
	id      int
	message string
}

func consumer(signal chan int, ctx context.Context) error {
	for {
		select {
		case <-signal:
			var msg Event

			mu.Lock()
			if len(Queue) > 0 {
				msg = Queue[0]
				Queue = Queue[1:]
			}

			fmt.Println(msg.id, msg.message)
			mu.Unlock()

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func producer(signal chan int, ctx context.Context) error {
	for {
		interval := time.Duration(rand.Int31n(10)) * time.Second

		select {
		case <-time.After(interval):
			mu.Lock()
			Queue = append(Queue, Event{id: ID, message: "Order_Created"})
			ID += 1
			mu.Unlock()

			signal <- 1
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
