package main

import (
	"flag"
	"fmt"
	"github.com/fatih/color"
	"net/http"
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
	flag.StringVar(&HttpMethod, "method", "", "HTTP Method to use while making the request")
	flag.StringVar(&Body, "body", "", "JSON Request Body for each Request")
	flag.Parse()
	if _, err := url2.ParseRequestURI(Endpoint); err != nil {
		fmt.Printf("Invalid Endpoint URL: %s\n", err)
		flag.PrintDefaults()
		os.Exit(-1)
	}

	if HttpMethod == "" {
		HttpMethod = http.MethodGet
	}

	switch HttpMethod {
	case http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodConnect, http.MethodOptions, http.MethodTrace:
	default:
		fmt.Printf("Invalid Http Method: %s\n", HttpMethod)
		os.Exit(-1)
	}

	println("USING:", runtime.NumCPU(), "CPUs")
	println("URL:", Endpoint)
	println("HTTP Method:", HttpMethod)
	println("Request Body:", Body)
	println("Total number of requests:", TotalReq)
	println("Parallel requests:", Concurrent)

	go LoadTest()

	ShowProgressBar()
	printResults()
}

func printResults() {
	color.Green("Succeeded Requests: %s", results[succeeded])
	color.Red("Failed Requests: %s", results[failed])
	color.Cyan("Requests/Second: %s", results[reqPerSecond])
	color.Cyan("Completed Load Testing In: %s", results[totalDuration])
}
