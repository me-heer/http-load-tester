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
	// Inputs
	TotalReq   int
	Endpoint   string
	Concurrent int

	// Accessed by other files to show results
	ReqProgress                   int
	AverageTimeToFirstByte        time.Duration
	TimeSpentMakingConnections    time.Duration
	NewConnectionsMade            atomic.Uint64
	AverageTimeTakenByEachRequest time.Duration
	FastestRequest                = time.Hour * 12
	SlowestRequest                time.Duration
	Elapsed                       time.Duration

	client  *http.Client
	start   time.Time
	results = make(map[string]string)
	workers = 0
)

const (
	failed       = "failed"
	succeeded    = "succeeded"
	reqPerSecond = "reqPerSecond"
)

type Response struct {
	response  *http.Response
	traceInfo ReqTraceInfo
}

type ReqTraceInfo struct {
	timeToFirstByte time.Duration
	timeToConnect   time.Duration
	total           time.Duration
}

func LoadTest() {
	start = time.Now()
	client = &http.Client{Transport: &http.Transport{MaxConnsPerHost: Concurrent, MaxIdleConns: Concurrent, MaxIdleConnsPerHost: Concurrent}}
	reqPool := make(chan *http.Request)
	respPool := make(chan *Response)
	go createRequestJobs(reqPool, Endpoint, TotalReq)
	go startRequestWorkers(reqPool, respPool, Concurrent)
	go evaluateResponses(respPool)
}

func createRequestJobs(reqPool chan<- *http.Request, url string, numberOfRequests int) {
	defer close(reqPool)
	for i := 0; i < numberOfRequests; i++ {
		r, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			panic(err)
		}
		reqPool <- r
	}
}

func evaluateResponses(responseChannel <-chan *Response) {
	var succeededCount int64
	var failedCount int64
	for ReqProgress < TotalReq {
		ar := <-responseChannel
		if ar.response.StatusCode == http.StatusOK {
			succeededCount++
		} else {
			failedCount++
		}
		ReqProgress++
		AverageTimeToFirstByte = (AverageTimeToFirstByte + ar.traceInfo.timeToFirstByte) / 2
		AverageTimeTakenByEachRequest = (AverageTimeTakenByEachRequest + ar.traceInfo.total) / 2
		if ar.traceInfo.total < FastestRequest {
			FastestRequest = ar.traceInfo.total
		}
		if ar.traceInfo.total > SlowestRequest {
			SlowestRequest = ar.traceInfo.total
		}
	}
	took := time.Since(start)
	Elapsed, _ = time.ParseDuration(fmt.Sprintf("%d", took.Nanoseconds()) + "ns")
	results[succeeded] = fmt.Sprintf("%d", succeededCount)
	results[failed] = fmt.Sprintf("%d", failedCount)
	requestsPerSecond := float64(succeededCount) / Elapsed.Seconds()
	results[reqPerSecond] = fmt.Sprintf("%f", requestsPerSecond)
}

func startRequestWorkers(requestChannel <-chan *http.Request, responseChannel chan<- *Response, maxConcurrentRequests int) {
	for i := 0; i < maxConcurrentRequests; i++ {
		go worker(requestChannel, responseChannel)
		workers++
	}
}

func worker(requestChannel <-chan *http.Request, responseChannel chan<- *Response) {
	for req := range requestChannel {
		var connect, reqStart time.Time
		var timeToFirstByte, timeToConnect time.Duration

		trace := &httptrace.ClientTrace{
			ConnectStart: func(network, addr string) { connect = time.Now() },
			ConnectDone: func(network, addr string, err error) {
				timeToConnect = time.Since(connect)
				TimeSpentMakingConnections += timeToConnect
			},

			GotConn: func(connInfo httptrace.GotConnInfo) {
				if !connInfo.Reused {
					NewConnectionsMade.Add(1)
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
			printResults()
			os.Exit(2)
		}
		_, err = io.ReadAll(resp.Body)
		if err != nil {
			return
		}

		totalTime := time.Since(reqStart)

		err = resp.Body.Close()
		if err != nil {
			return
		}

		traceInfo := ReqTraceInfo{
			timeToFirstByte: timeToFirstByte,
			timeToConnect:   timeToConnect,
			total:           totalTime,
		}
		ar := &Response{response: resp, traceInfo: traceInfo}

		responseChannel <- ar
	}
}
