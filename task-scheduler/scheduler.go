package scheduler

import (
	"container/heap"
	"context"
	"fmt"
	"sync"
	"time"
)

type TaskFunc func(ctx context.Context)

type node struct {
	task        TaskFunc
	scheduledAt time.Time
	id          int
}

type taskQueue struct {
	minHeap       *MinHeap
	mu            sync.Mutex
	notEmpty      *sync.Cond
	id            int
	canceledTasks map[int]bool
	isClosed      bool
	taskTimeOut   time.Duration
	earlyWakeup   chan int
	wg            sync.WaitGroup
}

type Scheduler interface {
	ScheduleAt(t time.Time, task TaskFunc) int
	ScheduleAfter(d time.Duration, task TaskFunc) int
	Cancel(taskID int) bool
	Close()
}

func NewTaskQueue(numWorkers int, taskTimeout time.Duration) Scheduler {
	minHeap := &MinHeap{}
	heap.Init(minHeap)

	tq := &taskQueue{
		minHeap:       minHeap,
		mu:            sync.Mutex{},
		taskTimeOut:   taskTimeout,
		canceledTasks: make(map[int]bool),
	}

	tq.notEmpty = sync.NewCond(&tq.mu)
	tq.earlyWakeup = make(chan int)

	tq.wg = sync.WaitGroup{}
	tq.wg.Add(numWorkers)

	tq.initProcessors(numWorkers)

	return tq
}

func (tq *taskQueue) initProcessors(numWorkers int) {
	for i := 0; i < numWorkers; i++ {
		go tq.taskProcessor(&tq.wg)
	}
}

func (tq *taskQueue) Enqueue(t time.Time, task TaskFunc) int {
	tq.mu.Lock()
	defer tq.mu.Unlock()

	if tq.isClosed {
		return -1
	}

	hasTask := false
	var curMin time.Time

	if tq.minHeap.Len() > 0 {
		hasTask = true
		curMin = (*tq.minHeap)[0].scheduledAt
	}

	heap.Push(tq.minHeap, node{task: task, scheduledAt: t, id: tq.id})
	taskID := tq.id

	tq.id = tq.id + 1

	tq.notEmpty.Signal()

	if !hasTask || t.Before(curMin) {
		select {
		case tq.earlyWakeup <- 1:
		default:
		}
	}

	tq.canceledTasks[taskID] = false

	return taskID
}

func (tq *taskQueue) Dequeue() (task TaskFunc, isCanceled, queueClosed bool) {
	tq.mu.Lock()
	defer tq.mu.Unlock()

	for {
		for tq.minHeap.Len() == 0 && !tq.isClosed {
			tq.notEmpty.Wait()
		}

		if tq.minHeap.Len() == 0 && tq.isClosed {
			return nil, false, true
		}

		next := (*tq.minHeap)[0]
		wait := time.Until(next.scheduledAt)

		if wait <= 0 {
			t := heap.Pop(tq.minHeap).(node)
			taskCanceled := tq.canceledTasks[t.id]
			delete(tq.canceledTasks, t.id)

			if taskCanceled {
				fmt.Println("task is canceled")

				return nil, true, false
			}

			return t.task, false, false
		}

		timer := time.NewTimer(wait)
		tq.mu.Unlock()

		select {
		case <-timer.C:
		case <-tq.earlyWakeup:
		}

		timer.Stop()
		tq.mu.Lock()
	}
}

func (tq *taskQueue) ScheduleAt(t time.Time, task TaskFunc) int {
	taskID := tq.Enqueue(t, task)
	fmt.Println("task has been scheduled, here is ur task id", taskID)
	return taskID
}

func (tq *taskQueue) ScheduleAfter(d time.Duration, task TaskFunc) int {
	t := time.Now().Add(d)
	taskID := tq.Enqueue(t, task)
	fmt.Println("task has been scheduled, here is ur task id", taskID)
	return taskID
}

func (tq *taskQueue) Close() {
	tq.mu.Lock()
	tq.isClosed = true
	tq.notEmpty.Broadcast()
	tq.mu.Unlock()

	close(tq.earlyWakeup)
	tq.wg.Wait()
}

func (tq *taskQueue) Cancel(taskID int) bool {
	tq.mu.Lock()
	defer tq.mu.Unlock()

	val, ok := tq.canceledTasks[taskID]
	if !ok {
		fmt.Println("task with id ", taskID, "does not exist or has already been processed")
		return false
	}

	if val {
		fmt.Println("task with id ", taskID, "has already been canceled")
		return false
	}

	tq.canceledTasks[taskID] = true
	fmt.Println("task canceled")
	return true
}

func (tq *taskQueue) taskProcessor(wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		task, isCanceled, queueClosed := tq.Dequeue()
		if queueClosed {
			return
		}

		if isCanceled || task == nil {
			continue
		}

		taskCtx, cancel := context.WithTimeout(context.Background(), tq.taskTimeOut)
		go func() {
			defer cancel()
			task(taskCtx)
		}()
	}
}

type MinHeap []node

func (h MinHeap) Len() int { return len(h) }
func (h MinHeap) Less(i, j int) bool {
	return h[i].scheduledAt.Before(h[j].scheduledAt)
}
func (h MinHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }

func (h *MinHeap) Push(x interface{}) {
	*h = append(*h, x.(node))
}

func (h *MinHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}
