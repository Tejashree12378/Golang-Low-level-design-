package main

import (
	"fmt"
	"sync"
	"time"
)

func main() {
	numWorkers := 3
	numJobs := 10
	executor(numWorkers, numJobs)
}

func worker(job <-chan int, results chan<- int, wg *sync.WaitGroup) {
	defer wg.Done()
	for j := range job {
		time.Sleep(time.Second)
		results <- j * j
	}
}

func jobScheduler(job chan<- int, numJobs int) {
	defer close(job)
	for i := 1; i <= numJobs; i++ {
		job <- i
	}
}

func executor(numWorkers int, numJobs int) {
	job := make(chan int)
	results := make(chan int)
	var wg = &sync.WaitGroup{}

	wg.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go worker(job, results, wg)
	}

	go jobScheduler(job, numJobs)

	go func() {
		wg.Wait()
		close(results)
	}()

	for i := range results {
		fmt.Println("processed: ", i)
	}
}
