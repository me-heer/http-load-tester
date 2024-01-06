package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptrace"
	"os"
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
	results           = make(map[string]string)
)

type ReqTraceInfo struct {
	timeToFirstByte time.Duration
	timeToConnect   time.Duration
	total           time.Duration
}

func PrintResults() {
	for key, value := range results {
		fmt.Printf("%s: %s\n", key, value)
	}
}

func LoadTest() {
	start = time.Now()
	traces = make([]ReqTraceInfo, TotalReq)
	client = &http.Client{Transport: &http.Transport{MaxConnsPerHost: concurrent, MaxIdleConns: concurrent, MaxIdleConnsPerHost: concurrent}}
	reqPool := make(chan *http.Request)
	respPool := make(chan *http.Response)
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
			if res.StatusCode == http.StatusOK {
				succeeded++
			} else {
				failed++
			}
			reqProgress++
		}
	}
	took := time.Since(start)
	averageTimeSpentPerRequest := took.Nanoseconds() / succeeded
	duration, err := time.ParseDuration(fmt.Sprintf("%d", averageTimeSpentPerRequest) + "ns")

	if err != nil {
		panic(err)
	}

	totalTime, err := time.ParseDuration(fmt.Sprintf("%d", took.Nanoseconds()) + "ns")
	if err != nil {
		panic(err)
	}
	results["Average time spent per request"] = fmt.Sprintf("%s", duration)
	results["Reused Connections"] = fmt.Sprintf("%d", reusedConnections.Load())
	results["Successful Requests"] = fmt.Sprintf("%d", succeeded)
	results["Failed Requests"] = fmt.Sprintf("%d", failed)
	results["Total time taken to complete load testing"] = fmt.Sprintf("%v", totalTime)
	reqPerSecond := totalTime.Seconds() / float64(succeeded)
	results["Requests/Second"] = fmt.Sprintf("%f", reqPerSecond)

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
