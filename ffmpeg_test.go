package main

import (
	"reflect"
	"testing"
)

func TestFFmpegArgs(t *testing.T) {
	tests := []struct {
		name      string
		overwrite bool
		verbose   bool
		mappings  []int
		want      []string
	}{
		{
			name: "default progress mode",
			want: []string{
				"-hide_banner", "-n", "-loglevel", "error", "-progress", "pipe:1", "-stats_period", "1",
				"-i", "https://example.com/playlist.m3u8",
				"-c", "copy", "-bsf:a", "aac_adtstoasc", "out.mp4",
			},
		},
		{
			name:      "overwrite verbose with mappings",
			overwrite: true,
			verbose:   true,
			mappings:  []int{2, 5},
			want: []string{
				"-hide_banner", "-y",
				"-i", "https://example.com/playlist.m3u8",
				"-map", "0:2", "-map", "0:5",
				"-c", "copy", "-bsf:a", "aac_adtstoasc", "out.mp4",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ffmpegArgs("https://example.com/playlist.m3u8", "out.mp4", tt.overwrite, tt.verbose, tt.mappings)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("args = %#v, want %#v", got, tt.want)
			}
		})
	}
}
