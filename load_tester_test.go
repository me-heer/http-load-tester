package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateRequestJobs(t *testing.T) {
	type args struct {
		reqPool  chan *http.Request
		url      string
		totalReq int
	}
	tests := []struct {
		name     string
		args     args
		expected int
	}{
		{
			name: "Test Request Pool of 100",
			args: args{
				reqPool:  make(chan *http.Request, 100),
				url:      "http://localhost:12345",
				totalReq: 100,
			},
			expected: 100,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			createRequestJobs(tt.args.reqPool, tt.args.url, tt.args.totalReq)
		})

		if len(tt.args.reqPool) != tt.expected {
			t.Fatalf("createRequestJobs did not create a request pool of %d.\n"+
				"Expected: %d Actual: %d", tt.expected, tt.args.totalReq, len(tt.args.reqPool))
		}
	}

}

func TestWorker(t *testing.T) {
	type args struct {
		requestChannel  <-chan *http.Request
		responseChannel chan *Response
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := fmt.Fprintln(w, "Hello, client")
		if err != nil {
			println("Could not write response from httptest server")
			log.Fatal(err.Error())
		}
	}))
	defer server.Close()
	client = http.DefaultClient

	reqChannelWithSampleReq := make(chan *http.Request, 10)
	createRequestJobs(reqChannelWithSampleReq, server.URL, 10)

	tests := []struct {
		name string
		args args
	}{
		{
			name: "Test worker with 10 request jobs",
			args: args{
				requestChannel:  reqChannelWithSampleReq,
				responseChannel: make(chan *Response, 10),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			worker(tt.args.requestChannel, tt.args.responseChannel)
		})
		if len(tt.args.responseChannel) != 10 {
			t.Fatalf("Not enough responses received in response channel."+"\n"+
				"Expected: %d, Actual: %d", 10, len(tt.args.responseChannel))
		}
		// TODO: 10 should be a constant declared somewhere
		for i := 0; i < 10; i++ {
			res := <-tt.args.responseChannel
			if res.response.StatusCode != 200 || res.traceInfo.total <= 0 {
				t.Fatalf("Did not get expected response.")
			}
		}
	}
}

func TestStartRequestWorkers(t *testing.T) {
	type args struct {
		requestChannel  <-chan *http.Request
		responseChannel chan *Response
	}

	tests := []struct {
		name string
		args args
	}{
		{
			name: "Test worker with 10 request jobs",
			args: args{
				requestChannel:  make(chan *http.Request, 10),
				responseChannel: make(chan *Response, 10),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			startRequestWorkers(tt.args.requestChannel, tt.args.responseChannel, 2)
		})
		if workers != 2 {
			t.Fatalf("Did not start enough workers")
		}
	}
}

func TestEvaluateResponses(t *testing.T) {
	type args struct {
		resPool chan *Response
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := fmt.Fprintln(w, "Hello, client")
		if err != nil {
			println("Could not write response from httptest server")
			log.Fatal(err.Error())
		}
	}))
	defer server.Close()
	client = http.DefaultClient

	TotalReq = 10
	reqChannelWithSampleReq := make(chan *http.Request, TotalReq)
	createRequestJobs(reqChannelWithSampleReq, server.URL, TotalReq)

	tests := []struct {
		name     string
		args     args
		expected int
	}{
		{
			name: "Test evaluateResponses with 10 responses",
			args: args{
				resPool: make(chan *Response, TotalReq),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			startRequestWorkers(reqChannelWithSampleReq, tt.args.resPool, TotalReq)
			evaluateResponses(tt.args.resPool)
		})
		if results[succeeded] != fmt.Sprint(TotalReq) && results[failed] != fmt.Sprint(0) {
			t.Fatal("Not enough requests passed")
		}
	}

}
