package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/httptrace"
	url2 "net/url"
	"os"
	"runtime"
	"sync/atomic"
	"time"
)

var (
	concurrent        int
	TotalReq          int
	Endpoint          string
	client            *http.Client
	start             time.Time
	reusedConnections atomic.Uint64
	traces            []ReqTraceInfo
	reqProgress       int
)

type ReqTraceInfo struct {
	timeToFirstByte time.Duration
	timeToConnect   time.Duration
	total           time.Duration
}

func init() {
	const (
		defaultNumberOfTotalRequests = 5
		defaultConcurrentRequests    = 1
	)
	flag.StringVar(&Endpoint, "url", "", "Endpoint URL for load testing")
	flag.IntVar(&TotalReq, "n", defaultNumberOfTotalRequests, "Total number of requests to make")
	flag.IntVar(&concurrent, "c", defaultConcurrentRequests, "Number of concurrent requests")
	traces = make([]ReqTraceInfo, TotalReq)
	client = &http.Client{Transport: &http.Transport{MaxConnsPerHost: concurrent, MaxIdleConns: concurrent, MaxIdleConnsPerHost: concurrent}}

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
}

func LoadTest() {
	reqPool := make(chan *http.Request)
	respPool := make(chan *http.Response)
	start = time.Now()
	go Dispatch(reqPool, Endpoint)
	go InitializeWorkerPool(reqPool, respPool)
	go Evaluate(respPool)
}

func Evaluate(responseChannel <-chan *http.Response) {
	var succeeded int64
	var failed int64
	for reqProgress < TotalReq {
		select {
		case res := <-responseChannel:
			if res.StatusCode == 200 {
				succeeded++
			} else {
				failed++
			}
			reqProgress++
		}
	}
	//println("EVALUATED")
	//took := time.Since(start)
	//averageTimeSpentPerRequest := took.Nanoseconds() / succeeded
	//duration, err := time.ParseDuration(fmt.Sprintf("%d", averageTimeSpentPerRequest) + "ns")
	//
	//if err != nil {
	//	panic(err)
	//}
	//
	//fmt.Printf("Average time spent per request: %s\n", duration)
	//
	//totalTime, err := time.ParseDuration(fmt.Sprintf("%d", took.Nanoseconds()) + "ns")
	//if err != nil {
	//	panic(err)
	//}
	//fmt.Printf("Reused Connections: %d\n", reusedConnections.Load())
	//fmt.Printf("Succeeded: %d\n", succeeded)
	//fmt.Printf("Failed: %d\n", failed)
	//fmt.Printf("Completed load testing in %s\n", totalTime)
	//reqPerSecond := totalTime.Seconds() / float64(succeeded)
	//fmt.Printf("Request/Second: %f\n", reqPerSecond)
}

func InitializeWorkerPool(requestChannel <-chan *http.Request, responseChannel chan<- *http.Response) {
	for i := 0; i < concurrent; i++ {
		go worker(requestChannel, responseChannel)
	}
}

func worker(requestChannel <-chan *http.Request, responseChannel chan<- *http.Response) {
	for req := range requestChannel {
		var connect, reqStart time.Time
		var timeToFirstByte, timeToConnect time.Duration

		trace := &httptrace.ClientTrace{
			ConnectStart: func(network, addr string) { connect = time.Now() },
			ConnectDone: func(network, addr string, err error) {
				timeToConnect = time.Since(connect)
			},

			GotConn: func(connInfo httptrace.GotConnInfo) {
				if connInfo.Reused {
					reusedConnections.Add(1)
				}
			},
			GotFirstResponseByte: func() {
				timeToFirstByte = time.Since(reqStart)
			},
		}
		reqStart = time.Now()
		req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))
		resp, err := client.Do(req)
		if err != nil {
			println(err.Error())
			os.Exit(2)
		}
		io.ReadAll(resp.Body)

		totalTime := time.Since(reqStart)

		err = resp.Body.Close()
		if err != nil {
			return
		}

		traces = append(traces, ReqTraceInfo{
			timeToFirstByte: timeToFirstByte,
			timeToConnect:   timeToConnect,
			total:           totalTime,
		})

		responseChannel <- resp
	}
}

func Dispatch(resPool chan *http.Request, url string) {
	defer close(resPool)
	for i := 0; i < TotalReq; i++ {
		r, err := http.NewRequest("GET", url, nil)
		if err != nil {
			panic(err)
		}
		resPool <- r
	}
}
