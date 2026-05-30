package main

import (
	"testing"
	"time"
)

func TestBestVideoStream(t *testing.T) {
	streams := []ffprobeStream{
		{Index: 0, CodecType: "video", BitRate: "500000", Width: 640, Height: 360},
		{Index: 1, CodecType: "audio", BitRate: "128000"},
		{Index: 2, CodecType: "video", BitRate: "1500000", Width: 1280, Height: 720},
		{Index: 3, CodecType: "video", BitRate: "1000000", Width: 1920, Height: 1080},
	}

	got, ok := bestVideoStream(streams)
	if !ok {
		t.Fatal("expected video stream")
	}
	if got.Index != 2 {
		t.Fatalf("index = %d, want 2", got.Index)
	}
}

func TestBestVideoStreamFallsBackToResolution(t *testing.T) {
	streams := []ffprobeStream{
		{Index: 4, CodecType: "video", Width: 640, Height: 360},
		{Index: 2, CodecType: "video", Width: 1920, Height: 1080},
	}

	got, ok := bestVideoStream(streams)
	if !ok {
		t.Fatal("expected video stream")
	}
	if got.Index != 2 {
		t.Fatalf("index = %d, want 2", got.Index)
	}
}

func TestAudioForVideoPrefersSameProgram(t *testing.T) {
	probe := ffprobeOutput{
		Streams: []ffprobeStream{
			{Index: 0, CodecType: "audio"},
			{Index: 1, CodecType: "video"},
			{Index: 2, CodecType: "audio"},
		},
		Programs: []ffprobeProgram{
			{Streams: []ffprobeStream{{Index: 1, CodecType: "video"}, {Index: 2, CodecType: "audio"}}},
		},
	}

	got, ok := audioForVideo(probe, 1)
	if !ok {
		t.Fatal("expected audio stream")
	}
	if got.Index != 2 {
		t.Fatalf("index = %d, want 2", got.Index)
	}
}

func TestBestDuration(t *testing.T) {
	selected := ffprobeStream{Index: 1, CodecType: "video", Duration: "12.5"}
	probe := ffprobeOutput{
		Streams: []ffprobeStream{
			{Index: 0, CodecType: "audio", Duration: "10"},
			selected,
		},
		Format: ffprobeFormat{Duration: "20"},
	}

	got := bestDuration(probe, &selected)
	if got != 12500*time.Millisecond {
		t.Fatalf("duration = %s, want 12.5s", got)
	}
}
