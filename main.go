package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"
)

var (
	nRequest int
	nWorker  int
)

var urls = []string{"http://helpify.dk", "http://helpify.dk", "http://helpify.dk"}

func main() {

	flag.IntVar(&nRequest, "r", len(urls), "Number of requesters")
	flag.IntVar(&nWorker, "w", 10, "Number of workers")

	work := make(chan URLRequest)

	results := make(chan URLResult, len(urls))

	for _, url := range urls {
		go URLRequester(work, url, fetchURL, results)
	}

	go printResults(results)

	NewBalancer(nWorker, nRequest).Balance(work)
}

func printResults(results <-chan URLResult) {
	for {
		res := <-results
		fmt.Println(res)
	}
}

func fetchURL(url string) URLResult {

	request, err := http.NewRequest("HEAD", url, nil)

	if err != nil {
		log.Fatal(err.Error())
		return URLResult{status: http.StatusNotFound, latency: -1}
	}

	client := http.DefaultClient

	t1 := time.Now()

	response, err := client.Do(request)

	elapsed := float64(time.Since(t1).Nanoseconds()) / 1e9

	if err != nil {
		log.Fatal(err.Error())
		return URLResult{status: http.StatusNotFound, latency: -1}
	}

	return URLResult{status: response.StatusCode, latency: elapsed}
}
