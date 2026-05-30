package main

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

type progressUpdateMsg struct {
	processed time.Duration
	speed     string
}

type downloadDoneMsg struct{}

type progressModel struct {
	spinner   spinner.Model
	progress  progress.Model
	processed time.Duration
	speed     string
	total     time.Duration
}

func newProgressModel(total time.Duration) progressModel {
	s := spinner.New()
	s.Spinner = spinner.Dot

	p := progress.New(progress.WithDefaultGradient(), progress.WithoutPercentage())
	p.Width = 32

	return progressModel{spinner: s, progress: p, total: total}
}

func (m progressModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m progressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case progressUpdateMsg:
		if msg.processed > 0 {
			m.processed = msg.processed
		}
		if msg.speed != "" {
			m.speed = msg.speed
		}
		return m, nil
	case downloadDoneMsg:
		return m, tea.Quit
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	default:
		return m, nil
	}
}

func (m progressModel) View() string {
	processed := "00:00:00"
	if m.processed > 0 {
		processed = formatDuration(m.processed)
	}

	speed := "unknown"
	if m.speed != "" && m.speed != "N/A" {
		speed = m.speed
	}

	if m.total > 0 {
		percent := clampPercent(float64(m.processed) / float64(m.total))
		return fmt.Sprintf(
			"Downloading %s %3.0f%%  %s / %s  speed %s\n",
			m.progress.ViewAs(percent),
			percent*100,
			processed,
			formatDuration(m.total),
			speed,
		)
	}

	return fmt.Sprintf("%s Downloading... processed %s, speed %s\n", m.spinner.View(), processed, speed)
}

func scanProgress(reader io.Reader, update func(progressUpdateMsg)) error {
	scanner := bufio.NewScanner(reader)

	var current progressUpdateMsg
	for scanner.Scan() {
		key, value, ok := strings.Cut(scanner.Text(), "=")
		if !ok {
			continue
		}

		switch key {
		case "out_time_ms":
			if processed, ok := parseOutTimeMS(value); ok {
				current.processed = processed
			}
		case "out_time":
			if processed, ok := parseOutTime(value); ok {
				current.processed = processed
			}
		case "speed":
			current.speed = value
		case "progress":
			update(current)
		}
	}

	return scanner.Err()
}

func parseOutTimeMS(value string) (time.Duration, bool) {
	microseconds, err := time.ParseDuration(value + "us")
	if err != nil {
		return 0, false
	}
	return microseconds, true
}

func parseOutTime(value string) (time.Duration, bool) {
	parts := strings.Split(value, ":")
	if len(parts) != 3 {
		return 0, false
	}

	hours, err := time.ParseDuration(parts[0] + "h")
	if err != nil {
		return 0, false
	}

	minutes, err := time.ParseDuration(parts[1] + "m")
	if err != nil {
		return 0, false
	}

	seconds, err := time.ParseDuration(parts[2] + "s")
	if err != nil {
		return 0, false
	}

	return hours + minutes + seconds, true
}

func formatDuration(duration time.Duration) string {
	total := int64(duration.Round(time.Second).Seconds())
	hours := total / 3600
	minutes := (total % 3600) / 60
	seconds := total % 60

	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
}

func clampPercent(percent float64) float64 {
	if percent < 0 {
		return 0
	}
	if percent > 1 {
		return 1
	}
	return percent
}
