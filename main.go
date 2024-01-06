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
	"strings"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	concurrent        int
	totalReq          int
	endpoint          string
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
	flag.StringVar(&endpoint, "url", "", "Endpoint URL for load testing")
	flag.IntVar(&totalReq, "n", defaultNumberOfTotalRequests, "Total number of requests to make")
	flag.IntVar(&concurrent, "c", defaultConcurrentRequests, "Number of concurrent requests")
	traces = make([]ReqTraceInfo, totalReq)
}

func main() {
	flag.Parse()
	client = &http.Client{Transport: &http.Transport{MaxConnsPerHost: concurrent, MaxIdleConns: concurrent, MaxIdleConnsPerHost: concurrent}}

	if _, err := url2.ParseRequestURI(endpoint); err != nil {
		fmt.Printf("Invalid Endpoint URL: %s\n", err)
		flag.PrintDefaults()
		os.Exit(-1)
	}

	println("USING:", runtime.NumCPU(), "CPUs")
	println("URL:", endpoint)
	println("Total number of requests:", totalReq)
	println("Parallel requests:", concurrent)

	runtime.GOMAXPROCS(runtime.NumCPU())
	reqPool := make(chan *http.Request)
	respPool := make(chan *http.Response)

	start = time.Now()
	go Dispatch(reqPool, endpoint)
	go InitializeWorkerPool(reqPool, respPool)
	go Evaluate(respPool)
	m := model{
		progress:        progress.New(progress.WithDefaultGradient()),
		currentRequests: 0,
	}

	if _, err := tea.NewProgram(m).Run(); err != nil {
		fmt.Println("Oh no!", err)
		os.Exit(1)
	}
}

func Evaluate(responseChannel <-chan *http.Response) {
	var succeeded int64
	var failed int64
	for reqProgress < totalReq {
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
	for i := 0; i < totalReq; i++ {
		r, err := http.NewRequest("GET", url, nil)
		if err != nil {
			panic(err)
		}
		resPool <- r
	}
}

const (
	padding  = 2
	maxWidth = 80
)

var helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).Render

type tickMsg time.Time

type model struct {
	progress        progress.Model
	currentRequests int
}

func (m model) Init() tea.Cmd {
	return tea.Cmd(checkProgress(&m))
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m, tea.Quit

	case tea.WindowSizeMsg:
		m.progress.Width = msg.Width - padding*2 - 4
		if m.progress.Width > maxWidth {
			m.progress.Width = maxWidth
		}
		return m, nil

	case tickMsg:
		if m.currentRequests >= totalReq {
			return m, tea.Quit
		}

		// Note that you can also use progress.Model.SetPercent to set the
		// percentage value explicitly, too.
		result := float64(reqProgress) / float64(totalReq)
		cmd := m.progress.SetPercent(result)
		return m, tea.Batch(checkProgress(&m), cmd)

	// FrameMsg is sent when the progress bar wants to animate itself
	case progress.FrameMsg:
		progressModel, cmd := m.progress.Update(msg)
		m.progress = progressModel.(progress.Model)
		return m, cmd

	default:
		return m, nil
	}
}

func (m model) View() string {
	pad := strings.Repeat(" ", padding)
	return "\n" +
		pad + m.progress.View() + "\n\n" +
		pad + helpStyle("Press any key to quit")
}

func checkProgress(m *model) tea.Cmd {
	m.currentRequests = reqProgress
	return tea.Tick(time.Millisecond*500, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}
