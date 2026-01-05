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
	index       int
}

type taskQueue struct {
	minHeap     *MinHeap
	mu          sync.Mutex
	notEmpty    *sync.Cond
	id          int
	taskMap     map[int]*node
	isClosed    bool
	taskTimeOut time.Duration
	earlyWakeup chan struct{}
	wg          sync.WaitGroup
	tasks       chan node
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
		minHeap:     minHeap,
		mu:          sync.Mutex{},
		taskTimeOut: taskTimeout,
		taskMap:     make(map[int]*node),
		tasks:       make(chan node),
	}

	tq.notEmpty = sync.NewCond(&tq.mu)
	tq.earlyWakeup = make(chan struct{}, 1)

	tq.wg = sync.WaitGroup{}

	tq.initProcessors(numWorkers)

	return tq
}

func (tq *taskQueue) initProcessors(numWorkers int) {
	tq.wg.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go tq.taskProcessor(&tq.wg)
	}

	tq.wg.Add(1)
	go tq.Dequeue()
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

	newTask := &node{task: task, scheduledAt: t, id: tq.id}

	heap.Push(tq.minHeap, newTask)
	taskID := tq.id

	tq.id = tq.id + 1

	tq.notEmpty.Signal()

	if !hasTask || t.Before(curMin) {
		select {
		case tq.earlyWakeup <- struct{}{}:
		default:
		}
	}

	tq.taskMap[taskID] = newTask

	return taskID
}

func (tq *taskQueue) Dequeue() {
	defer tq.wg.Done()

	timer := time.NewTimer(time.Hour)
	defer timer.Stop()

	for {
		tq.mu.Lock()

		for tq.minHeap.Len() == 0 && !tq.isClosed {
			tq.notEmpty.Wait()
		}

		if tq.minHeap.Len() == 0 && tq.isClosed {
			close(tq.tasks)
			tq.mu.Unlock()
			return
		}

		next := (*tq.minHeap)[0]
		wait := time.Until(next.scheduledAt)
		tasks := make([]*node, 0)

		if wait > 0 {
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}

			timer.Reset(wait)
			tq.mu.Unlock()

			select {
			case <-timer.C:
			case <-tq.earlyWakeup:
			}

			continue
		}

		now := time.Now()
		for tq.minHeap.Len() > 0 && !(*tq.minHeap)[0].scheduledAt.After(now) {
			t := heap.Pop(tq.minHeap).(*node)
			fmt.Println("dequeued", t.id)

			if _, ok := tq.taskMap[t.id]; ok {
				tasks = append(tasks, t)
				delete(tq.taskMap, t.id)
			}
		}

		tq.mu.Unlock()

		for _, t := range tasks {
			tq.tasks <- *t
		}
	}
}

func (tq *taskQueue) ScheduleAt(t time.Time, task TaskFunc) int {
	taskID := tq.Enqueue(t, task)
	return taskID
}

func (tq *taskQueue) ScheduleAfter(d time.Duration, task TaskFunc) int {
	t := time.Now().Add(d)
	taskID := tq.Enqueue(t, task)
	return taskID
}

func (tq *taskQueue) Close() {
	tq.mu.Lock()
	tq.isClosed = true
	tq.notEmpty.Broadcast()
	tq.mu.Unlock()

	tq.wg.Wait()
}

func (tq *taskQueue) Cancel(taskID int) bool {
	tq.mu.Lock()
	defer tq.mu.Unlock()

	val, ok := tq.taskMap[taskID]
	if !ok {
		return false
	}

	delete(tq.taskMap, taskID)
	fmt.Println("deleted", heap.Remove(tq.minHeap, val.index))
	return true
}

func (tq *taskQueue) taskProcessor(wg *sync.WaitGroup) {
	defer wg.Done()
	for j := range tq.tasks {
		taskCtx, cancel := context.WithTimeout(context.Background(), tq.taskTimeOut)
		go func() {
			defer cancel()
			j.task(taskCtx)
		}()
	}
}

type MinHeap []*node

func (h MinHeap) Len() int { return len(h) }
func (h MinHeap) Less(i, j int) bool {
	return h[i].scheduledAt.Before(h[j].scheduledAt)
}
func (h MinHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index = i
	h[j].index = j
}

func (h *MinHeap) Push(x interface{}) {
	n := len(*h)
	item := x.(*node)
	item.index = n
	*h = append(*h, x.(*node))
}

func (h *MinHeap) Pop() interface{} {
	old := *h
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.index = -1
	*h = old[0 : n-1]
	return item
}
