package main

import (
	"flag"
	"fmt"
	"github.com/fatih/color"
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
	flag.IntVar(&Concurrent, "c", defaultConcurrentRequests, "Number of Concurrent requests")
	flag.Parse()
	if _, err := url2.ParseRequestURI(Endpoint); err != nil {
		fmt.Printf("Invalid Endpoint URL: %s\n", err)
		flag.PrintDefaults()
		os.Exit(-1)
	}

	println("USING:", runtime.NumCPU(), "CPUs")
	println("URL:", Endpoint)
	println("Total number of requests:", TotalReq)
	println("Parallel requests:", Concurrent)

	go LoadTest()

	ShowProgressBar()

	color.Green("Succeeded Requests: %s", results[succeeded])
	color.Red("Failed Requests: %s", results[failed])
	color.Cyan("Requests/Second: %s", results[reqPerSecond])
}
