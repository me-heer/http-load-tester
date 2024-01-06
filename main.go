package main

import (
	"fmt"
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"os"
	"runtime"
	"strings"
	"time"
)

func main() {

	runtime.GOMAXPROCS(runtime.NumCPU())

	go LoadTest()
	m := model{
		progress:        progress.New(progress.WithDefaultGradient()),
		currentRequests: 0,
	}

	if _, err := tea.NewProgram(m).Run(); err != nil {
		fmt.Println("Oh no!", err)
		os.Exit(1)
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
		if m.currentRequests >= TotalReq {
			return m, tea.Quit
		}

		// Note that you can also use progress.Model.SetPercent to set the
		// percentage value explicitly, too.
		result := float64(reqProgress) / float64(TotalReq)
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
