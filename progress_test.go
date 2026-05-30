package main

import (
	"strings"
	"testing"
	"time"
)

func TestScanProgress(t *testing.T) {
	input := strings.NewReader(strings.Join([]string{
		"out_time_ms=1500000",
		"speed=2.5x",
		"progress=continue",
		"out_time=00:00:03.000000",
		"speed=3.0x",
		"progress=end",
	}, "\n"))

	var updates []progressUpdateMsg
	err := scanProgress(input, func(update progressUpdateMsg) {
		updates = append(updates, update)
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(updates) != 2 {
		t.Fatalf("updates = %d, want 2", len(updates))
	}
	if updates[0].processed != 1500*time.Millisecond {
		t.Fatalf("first processed = %s, want 1.5s", updates[0].processed)
	}
	if updates[0].speed != "2.5x" {
		t.Fatalf("first speed = %q, want 2.5x", updates[0].speed)
	}
	if updates[1].processed != 3*time.Second {
		t.Fatalf("second processed = %s, want 3s", updates[1].processed)
	}
	if updates[1].speed != "3.0x" {
		t.Fatalf("second speed = %q, want 3.0x", updates[1].speed)
	}
}

func TestFormatDuration(t *testing.T) {
	got := formatDuration(90*time.Minute + 3*time.Second)
	if got != "01:30:03" {
		t.Fatalf("formatDuration = %q, want 01:30:03", got)
	}
}

func TestClampPercent(t *testing.T) {
	tests := []struct {
		in   float64
		want float64
	}{
		{in: -0.1, want: 0},
		{in: 0.4, want: 0.4},
		{in: 1.2, want: 1},
	}

	for _, tt := range tests {
		if got := clampPercent(tt.in); got != tt.want {
			t.Fatalf("clampPercent(%v) = %v, want %v", tt.in, got, tt.want)
		}
	}
}
