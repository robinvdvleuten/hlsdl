package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os/exec"
	"strconv"
	"time"
)

type probeResult struct {
	mappings []int
	duration time.Duration
}

type ffprobeOutput struct {
	Streams  []ffprobeStream  `json:"streams"`
	Programs []ffprobeProgram `json:"programs"`
	Format   ffprobeFormat    `json:"format"`
}

type ffprobeStream struct {
	Index     int    `json:"index"`
	CodecType string `json:"codec_type"`
	BitRate   string `json:"bit_rate"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	Duration  string `json:"duration"`
}

type ffprobeProgram struct {
	Streams []ffprobeStream `json:"streams"`
}

type ffprobeFormat struct {
	Duration string `json:"duration"`
}

func probeInput(parent context.Context, ffprobe, inputURL string) probeResult {
	ctx, cancel := context.WithTimeout(parent, 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, ffprobe,
		"-v", "error",
		"-print_format", "json",
		"-show_streams",
		"-show_programs",
		"-show_format",
		inputURL,
	)

	var output bytes.Buffer
	cmd.Stdout = &output
	if err := cmd.Run(); err != nil {
		return probeResult{}
	}

	var probe ffprobeOutput
	if err := json.Unmarshal(output.Bytes(), &probe); err != nil {
		return probeResult{}
	}

	video, ok := bestVideoStream(probe.Streams)
	if !ok {
		return probeResult{duration: bestDuration(probe, nil)}
	}

	mappings := []int{video.Index}
	if audio, ok := audioForVideo(probe, video.Index); ok {
		mappings = append(mappings, audio.Index)
	}

	return probeResult{
		mappings: mappings,
		duration: bestDuration(probe, &video),
	}
}

func bestVideoStream(streams []ffprobeStream) (ffprobeStream, bool) {
	var best ffprobeStream
	found := false
	for _, stream := range streams {
		if stream.CodecType != "video" {
			continue
		}

		if !found || betterVideo(stream, best) {
			best = stream
			found = true
		}
	}

	return best, found
}

func betterVideo(candidate, current ffprobeStream) bool {
	candidateBitrate := parseInt(candidate.BitRate)
	currentBitrate := parseInt(current.BitRate)
	if candidateBitrate != currentBitrate {
		return candidateBitrate > currentBitrate
	}

	candidatePixels := candidate.Width * candidate.Height
	currentPixels := current.Width * current.Height
	if candidatePixels != currentPixels {
		return candidatePixels > currentPixels
	}

	return candidate.Index < current.Index
}

func audioForVideo(probe ffprobeOutput, videoIndex int) (ffprobeStream, bool) {
	for _, program := range probe.Programs {
		hasVideo := false
		for _, stream := range program.Streams {
			if stream.Index == videoIndex {
				hasVideo = true
				break
			}
		}

		if !hasVideo {
			continue
		}

		for _, stream := range program.Streams {
			if stream.CodecType == "audio" {
				return stream, true
			}
		}
	}

	for _, stream := range probe.Streams {
		if stream.CodecType == "audio" {
			return stream, true
		}
	}

	return ffprobeStream{}, false
}

func bestDuration(probe ffprobeOutput, selectedVideo *ffprobeStream) time.Duration {
	if selectedVideo != nil {
		if duration := parseSecondsDuration(selectedVideo.Duration); duration > 0 {
			return duration
		}
	}

	var best time.Duration
	for _, stream := range probe.Streams {
		if duration := parseSecondsDuration(stream.Duration); duration > best {
			best = duration
		}
	}

	if best > 0 {
		return best
	}

	return parseSecondsDuration(probe.Format.Duration)
}

func parseInt(value string) int {
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return parsed
}

func parseSecondsDuration(value string) time.Duration {
	if value == "" || value == "N/A" {
		return 0
	}

	seconds, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0
	}

	return time.Duration(seconds * float64(time.Second))
}
