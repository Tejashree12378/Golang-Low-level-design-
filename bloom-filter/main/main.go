package main

import (
	"bloom-filter"
	"fmt"
	"sync"
	"time"
)

func main() {
	N := 1000000
	k := 4

	bf := filter.NewFilter(N, k)

	start := time.Now()

	for i := 0; i < N; i++ {
		bf.Add(fmt.Sprintf("item-%d", i))
	}

	elapsed := time.Since(start)
	fmt.Println("total time:", elapsed)

	start = time.Now()
	found := 0
	for i := 0; i < N; i++ {
		if bf.MightContain(fmt.Sprintf("item-%d", i)) {
			found++
		}
	}
	fmt.Println("total time:", time.Since(start))
	fmt.Println("found :", found)

	start = time.Now()
	notFound := 0
	for i := 0; i < N; i++ {
		if bf.MightContain(fmt.Sprintf("item1-%d", i)) {
			notFound++
		}
	}
	fmt.Println("total time:", time.Since(start))
	fmt.Println("notFound:", notFound)
	fmt.Println("false positives:", N-notFound)

	concurrencyTest(bf)
}

func concurrencyTest(bf filter.BloomFilter) {
	wg := sync.WaitGroup{}
	workers := 8

	start := time.Now()

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for i := 0; i < 200000; i++ {
				key := fmt.Sprintf("%d-%d", id, i)
				bf.Add(key)
				bf.MightContain(key)
			}
		}(w)
	}

	wg.Wait()

	fmt.Println("total time:", time.Since(start))
}
