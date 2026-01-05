package main

import (
	"context"
	"fmt"
	"math/rand"
	"task-scheduler"
	"time"
)

func main() {
	tq := scheduler.NewTaskQueue(5, time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	scheduleSimulation(ctx, tq)
	tq.Close()

	time.Sleep(20 * time.Second)
}

var task scheduler.TaskFunc = func(ctx context.Context) {
	select {
	case <-ctx.Done():
		return
	case <-time.After(time.Duration(rand.Int31n(50)) * time.Millisecond):
		return
	}
}

func scheduleSimulation(ctx context.Context, tq scheduler.Scheduler) {
	id := tq.ScheduleAt(time.Now().Add(10*time.Second), task)
	fmt.Println("ScheduleAt ", id, time.Now().Add(10*time.Second).String())

	for {
		t := time.Duration(rand.Int31n(1000)) * time.Millisecond
		t2 := time.Duration(rand.Int31n(1000)) * time.Millisecond
		select {
		case <-ctx.Done():
			return
		case <-time.After(t):
			id := tq.ScheduleAt(time.Now().Add(t), task)
			fmt.Println("ScheduleAt ", id, time.Now().Add(t).String())
		case <-time.After(t2):
			id := tq.ScheduleAfter(t2, task)
			fmt.Println("ScheduleAfter ", id, time.Now().Add(t).String())
		}
	}
}
