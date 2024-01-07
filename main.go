package main

import (
	"flag"
	"fmt"
	url2 "net/url"
	"os"
	"runtime"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	const (
		defaultNumberOfTotalRequests = 5
		defaultConcurrentRequests    = 1
	)
	flag.StringVar(&Endpoint, "url", "", "Endpoint URL for load testing")
	flag.IntVar(&TotalReq, "n", defaultNumberOfTotalRequests, "Total number of requests to make")
	flag.IntVar(&concurrent, "c", defaultConcurrentRequests, "Number of concurrent requests")
	flag.Parse()
	if _, err := url2.ParseRequestURI(Endpoint); err != nil {
		fmt.Printf("Invalid Endpoint URL: %s\n", err)
		flag.PrintDefaults()
		os.Exit(-1)
	}

	println("USING:", runtime.NumCPU(), "CPUs")
	println("URL:", Endpoint)
	println("Total number of requests:", TotalReq)
	println("Parallel requests:", concurrent)

	go LoadTest()
	ShowProgressBar()
	ShowResultsTable()
}
