package main

import (
	"net/http"
	"testing"
)

func TestLoadTest(t *testing.T) {
	tests := []struct {
		name string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			LoadTest()
		})
	}
}

func Test_createRequestJobs(t *testing.T) {
	type args struct {
		resPool chan *http.Request
		url     string
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			createRequestJobs(tt.args.resPool, tt.args.url)
		})
	}
}

func Test_evaluateResponses(t *testing.T) {
	type args struct {
		responseChannel <-chan *Response
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluateResponses(tt.args.responseChannel)
		})
	}
}

func Test_startRequestWorkers(t *testing.T) {
	type args struct {
		requestChannel  <-chan *http.Request
		responseChannel chan<- *Response
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			startRequestWorkers(tt.args.requestChannel, tt.args.responseChannel)
		})
	}
}

func Test_worker(t *testing.T) {
	type args struct {
		requestChannel  <-chan *http.Request
		responseChannel chan<- *Response
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			worker(tt.args.requestChannel, tt.args.responseChannel)
		})
	}
}
