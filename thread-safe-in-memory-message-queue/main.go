package main

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

type Message struct {
	id      int
	payload string
}

type Queue interface {
	Enqueue(msg Message) error
	Dequeue() (Message, error)
	Close()
}

type InMemoryQueue struct {
	buf      []Message
	head     int
	tail     int
	size     int
	capacity int
	closed   bool
	mu       *sync.Mutex
	notFull  *sync.Cond
	notEmpty *sync.Cond
}

func NewInMemoryQueue(capacity int) Queue {
	q := &InMemoryQueue{
		buf:      make([]Message, capacity),
		head:     0,
		tail:     0,
		size:     0,
		capacity: capacity,
		closed:   false,
		mu:       &sync.Mutex{},
	}

	q.notFull = sync.NewCond(q.mu)
	q.notEmpty = sync.NewCond(q.mu)
	return q
}

func (q *InMemoryQueue) Enqueue(msg Message) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	for q.size == q.capacity && !q.closed {
		q.notFull.Wait()
	}

	if q.closed {
		return errors.New("queue is closed")
	}

	q.buf[q.tail] = msg
	q.tail = (q.tail + 1) % q.capacity
	q.size += 1

	q.notEmpty.Signal()

	return nil
}

func (q *InMemoryQueue) Dequeue() (Message, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	for q.size == 0 && !q.closed {
		q.notEmpty.Wait()
	}

	if q.size == 0 && q.closed {
		return Message{}, errors.New("queue is closed")
	}

	msg := q.buf[q.head]
	q.head = (q.head + 1) % q.capacity
	q.size -= 1

	q.notFull.Signal()

	return msg, nil
}

func (q *InMemoryQueue) Close() {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.closed {
		return
	}

	q.closed = true

	q.notFull.Broadcast()
	q.notEmpty.Broadcast()
}

func producer(ctx context.Context, q Queue, wg *sync.WaitGroup) {
	defer wg.Done()
	id := 0
	for {
		select {
		case <-ctx.Done():
			return
		default:
			msg := Message{id: id, payload: "random"}
			if err := q.Enqueue(msg); err != nil {
				fmt.Println(err)
				return
			}

			id++
			time.Sleep(time.Duration(rand.Intn(300)) * time.Millisecond)
		}
	}
}

func consumer(ctx context.Context, q Queue, wg *sync.WaitGroup) {
	defer wg.Done()
	id := 0
	for {
		select {
		case <-ctx.Done():
			return
		default:
			msg, err := q.Dequeue()
			if err != nil {
				fmt.Println(err)
				return
			}

			fmt.Println("message received ", msg.id, msg.payload)
			id++
			time.Sleep(time.Duration(rand.Intn(500)) * time.Millisecond)
		}
	}
}

const (
	NUM_WORKERS = 1
	TIMEOUT     = 20 * time.Second
	CAPACITY    = 10
)

func main() {
	q := NewInMemoryQueue(CAPACITY)
	wg := &sync.WaitGroup{}

	wg.Add(NUM_WORKERS * 2)

	ctx, cancel := context.WithTimeout(context.Background(), TIMEOUT)
	defer cancel()

	for i := 0; i < NUM_WORKERS; i++ {
		go consumer(ctx, q, wg)
		go producer(ctx, q, wg)
	}

	<-ctx.Done()
	q.Close()
	wg.Wait()
}
