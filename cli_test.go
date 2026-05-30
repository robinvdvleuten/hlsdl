package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestParseBatchLine(t *testing.T) {
	tests := []struct {
		name       string
		line       string
		wantURL    string
		wantOutput string
		wantErr    bool
	}{
		{
			name:    "url only",
			line:    "https://example.com/video/SEG.m3u8",
			wantURL: "https://example.com/video/SEG.m3u8",
		},
		{
			name:       "url and output",
			line:       "https://example.com/video/SEG.m3u8 output.mp4",
			wantURL:    "https://example.com/video/SEG.m3u8",
			wantOutput: "output.mp4",
		},
		{
			name:       "output path with spaces",
			line:       "https://example.com/video/SEG.m3u8 My Episode 01.mp4",
			wantURL:    "https://example.com/video/SEG.m3u8",
			wantOutput: "My Episode 01.mp4",
		},
		{
			name:    "empty",
			line:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotURL, gotOutput, err := parseBatchLine(tt.line)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if gotURL != tt.wantURL {
				t.Fatalf("url = %q, want %q", gotURL, tt.wantURL)
			}
			if gotOutput != tt.wantOutput {
				t.Fatalf("output = %q, want %q", gotOutput, tt.wantOutput)
			}
		})
	}
}

func TestValidateURL(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantErr bool
	}{
		{name: "http", raw: "http://example.com/playlist.m3u8"},
		{name: "https", raw: "https://example.com/playlist.m3u8"},
		{name: "ftp", raw: "ftp://example.com/playlist.m3u8", wantErr: true},
		{name: "not a url", raw: "playlist.m3u8", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateURL(tt.raw)
			if tt.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateOutput(t *testing.T) {
	dir := t.TempDir()
	existing := filepath.Join(dir, "existing.mp4")
	if err := touch(existing); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name      string
		output    string
		overwrite bool
		wantErr   string
	}{
		{name: "mp4", output: filepath.Join(dir, "new.mp4")},
		{name: "uppercase extension", output: filepath.Join(dir, "new.MP4")},
		{name: "wrong extension", output: filepath.Join(dir, "new.mov"), wantErr: ".mp4"},
		{name: "existing without overwrite", output: existing, wantErr: "already exists"},
		{name: "existing with overwrite", output: existing, overwrite: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := validateOutput(tt.output, tt.overwrite)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error = %q, want substring %q", err, tt.wantErr)
			}
		})
	}
}

func TestDeriveOutputPath(t *testing.T) {
	dir := t.TempDir()
	seen := make(map[string]int)

	tests := []struct {
		name    string
		rawURL  string
		want    string
		wantErr bool
	}{
		{
			name:   "generic playlist uses parent",
			rawURL: "https://example.com/show/episode-01/SEG.m3u8",
			want:   filepath.Join(dir, "episode-01.mp4"),
		},
		{
			name:   "playlist filename",
			rawURL: "https://example.com/show/episode-02.m3u8?token=secret",
			want:   filepath.Join(dir, "episode-02.mp4"),
		},
		{
			name:   "duplicate derived name is numbered",
			rawURL: "https://example.com/show/episode-01/SEG.m3u8",
			want:   filepath.Join(dir, "episode-01-2.mp4"),
		},
		{
			name:   "unsafe characters sanitized",
			rawURL: "https://example.com/show/episode:03/SEG.m3u8",
			want:   filepath.Join(dir, "episode-03.mp4"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := deriveOutputPath(tt.rawURL, dir, seen)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("output = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBatchJobs(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "downloads.txt")
	content := strings.Join([]string{
		"# comment",
		"",
		"https://example.com/show/episode-01/SEG.m3u8",
		"https://example.com/show/episode-02/SEG.m3u8 explicit output.mp4",
	}, "\n")
	if err := writeFile(input, content); err != nil {
		t.Fatal(err)
	}

	cli := CLI{Input: input, OutputDir: dir}
	got, err := cli.batchJobs(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []downloadJob{
		{url: "https://example.com/show/episode-01/SEG.m3u8", output: filepath.Join(dir, "episode-01.mp4")},
		{url: "https://example.com/show/episode-02/SEG.m3u8", output: "explicit output.mp4"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("jobs = %#v, want %#v", got, want)
	}
}

func touch(path string) error {
	return writeFile(path, "")
}

func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}
