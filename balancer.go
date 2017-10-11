package main

import (
	"bytes"
	"container/heap"
	"fmt"
	"math/rand"
	"time"
)

type URLRequest struct {
	fn  func(url string) URLResult // The operation to perform
	url string                     // The URL to fetch
	c   chan URLResult             // The channel to return the result
}

type URLResult struct {
	status  int     // Status code of the request
	latency float64 // Time to perform the request
}

func (res URLResult) String() string {
	return fmt.Sprintf("StatusCode: %v, Latency: %v", res.status, res.latency)
}

func workFn() int {
	n := rand.Int63n(int64(time.Second))
	time.Sleep(time.Duration(rand.Int63n(5e9)))
	return int(n)
}

func URLRequester(work chan<- URLRequest, url string, fn func(string) URLResult, results chan<- URLResult) {

	c := make(chan URLResult)

	for {
		work <- URLRequest{fn: fn, url: url, c: c}
		res := <-c
		results <- res
		// Wait 5 seconds
		time.Sleep(time.Duration(1e9 * 5))
	}
}

type Worker struct {
	requests chan URLRequest // work to do (buffered channel)
	pending  int             // count of pending tasks
	index    int             // index in the heap
}

func (w *Worker) work(done chan *Worker) {
	for {
		req := <-w.requests
		req.c <- req.fn(req.url)
		done <- w
	}
}

type Pool []*Worker

func (p Pool) Len() int {
	return len(p)
}

func (p Pool) Less(i, j int) bool {
	return p[i].pending < p[j].pending
}

func (p Pool) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
	p[i].index = i
	p[j].index = j
}

func (p *Pool) Push(x interface{}) {
	a := *p
	*p = append(a, x.(*Worker))
}

func (p *Pool) Pop() interface{} {
	old := *p
	n := len(old)
	x := old[n-1]
	*p = old[0 : n-1]

	return x
}

type Balancer struct {
	pool Pool
	done chan *Worker
}

func NewBalancer(nWorker int, nRequester int) *Balancer {
	workers := make(Pool, 0, nWorker)

	done := make(chan *Worker)

	for i := 0; i <= nWorker; i++ {
		w := &Worker{
			requests: make(chan URLRequest, nRequester),
			pending:  0,
		}
		go w.work(done)
		heap.Push(&workers, w)
	}

	return &Balancer{
		pool: workers,
		done: done,
	}
}

func (b Balancer) Balance(work chan URLRequest) {
	for {
		select {
		case r := <-work:
			b.dispatch(r)
		case w := <-b.done:
			b.complete(w)

		}
	}
}

func (b Balancer) dispatch(r URLRequest) {
	w := heap.Pop(&b.pool).(*Worker)
	w.pending++
	w.requests <- r
	heap.Push(&b.pool, w)
	//fmt.Println(b)
}

func (b Balancer) complete(w *Worker) {
	w = heap.Remove(&b.pool, w.index).(*Worker)
	w.pending--
	heap.Push(&b.pool, w)
	//fmt.Println(b)
}

func (b Balancer) String() string {
	var buf bytes.Buffer

	sum := 0

	for _, w := range b.pool {
		sum += w.pending
		buf.WriteString(fmt.Sprintf("%2d", w.pending))
	}

	buf.WriteString(fmt.Sprintf("%8.2f", float64(sum)/float64(len(b.pool))))

	return buf.String()
}
