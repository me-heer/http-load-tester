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
	// Accessed by other files to show results
	TotalReq                      int
	Endpoint                      string
	ReqProgress                   int
	AverageTimeToFirstByte        time.Duration
	TimeSpentMakingConnections    time.Duration
	NewConnectionsMade            atomic.Uint64
	AverageTimeTakenByEachRequest time.Duration
	FastestRequest                = time.Hour * 12
	SlowestRequest                time.Duration
	Elapsed                       time.Duration

	concurrent        int
	client            *http.Client
	start             time.Time
	reusedConnections atomic.Uint64
	results           = make(map[string]string)
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
	client = &http.Client{Transport: &http.Transport{MaxConnsPerHost: concurrent, MaxIdleConns: concurrent, MaxIdleConnsPerHost: concurrent}}
	reqPool := make(chan *http.Request)
	respPool := make(chan *Response)
	go createRequestJobs(reqPool, Endpoint)
	go startRequestWorkers(reqPool, respPool)
	go evaluateResponses(respPool)
}

func createRequestJobs(resPool chan *http.Request, url string) {
	defer close(resPool)
	for i := 0; i < TotalReq; i++ {
		r, err := http.NewRequest("GET", url, nil)
		if err != nil {
			panic(err)
		}
		resPool <- r
	}
}

func evaluateResponses(responseChannel <-chan *Response) {
	var succeeded int64
	var failed int64
	for ReqProgress < TotalReq {
		select {
		case ar := <-responseChannel:
			if ar.response.StatusCode == http.StatusOK {
				succeeded++
			} else {
				failed++
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
	}
	took := time.Since(start)
	Elapsed, _ = time.ParseDuration(fmt.Sprintf("%d", took.Nanoseconds()) + "ns")
	results["Reused Connections"] = fmt.Sprintf("%d", reusedConnections.Load())
	results["Successful Requests"] = fmt.Sprintf("%d", succeeded)
	results["Failed Requests"] = fmt.Sprintf("%d", failed)
	reqPerSecond := float64(succeeded) / Elapsed.Seconds()
	results["Requests/Second"] = fmt.Sprintf("%f", reqPerSecond)
}

func startRequestWorkers(requestChannel <-chan *http.Request, responseChannel chan<- *Response) {
	for i := 0; i < concurrent; i++ {
		go worker(requestChannel, responseChannel)
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
				if connInfo.Reused {
					reusedConnections.Add(1)
				} else {
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
