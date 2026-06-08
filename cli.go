package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type downloadJob struct {
	url    string
	output string
}

func (cli CLI) Run(parent context.Context, stdout, stderr io.Writer) error {
	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		return errors.New("ffmpeg was not found on PATH")
	}

	ffprobe, err := exec.LookPath("ffprobe")
	if err != nil {
		return errors.New("ffprobe was not found on PATH")
	}

	ctx := parent
	cancel := func() {}
	if cli.Timeout > 0 {
		ctx, cancel = context.WithTimeout(parent, cli.Timeout)
	}
	defer cancel()

	jobs, err := cli.downloadJobs()
	if err != nil {
		return err
	}

	if len(jobs) == 1 {
		return cli.downloadOne(ctx, ffmpeg, ffprobe, jobs[0], stdout, stderr)
	}

	return cli.downloadBatch(ctx, ffmpeg, ffprobe, jobs, stdout, stderr)
}

func (cli CLI) downloadOne(ctx context.Context, ffmpeg, ffprobe string, job downloadJob, stdout, stderr io.Writer) error {
	probe := probeInput(ctx, ffprobe, job.url)
	args := ffmpegArgs(job.url, job.output, cli.Overwrite, cli.Verbose, probe.mappings)
	cmd := exec.CommandContext(ctx, ffmpeg, args...)

	if err := cli.runDownload(ctx, cmd, stdout, stderr, probe.duration); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("download timed out after %s", cli.Timeout)
		}
		return err
	}

	if !cli.Quiet {
		_, _ = fmt.Fprintf(stdout, "saved %s\n", job.output)
	}

	return nil
}

func (cli CLI) downloadBatch(ctx context.Context, ffmpeg, ffprobe string, jobs []downloadJob, stdout, stderr io.Writer) error {
	var failures []string

	for index, job := range jobs {
		if !cli.Quiet {
			_, _ = fmt.Fprintf(stdout, "[%d/%d] %s\n", index+1, len(jobs), job.output)
		}

		if err := cli.downloadOne(ctx, ffmpeg, ffprobe, job, stdout, stderr); err != nil {
			failures = append(failures, fmt.Sprintf("- %s: %v", job.url, err))
			_, _ = fmt.Fprintf(stderr, "error: %s: %v\n", job.url, err)
		}
	}

	if len(failures) == 0 {
		if !cli.Quiet {
			_, _ = fmt.Fprintf(stdout, "completed %d/%d downloads\n", len(jobs), len(jobs))
		}
		return nil
	}

	_, _ = fmt.Fprintf(stderr, "completed %d/%d downloads\nfailed:\n%s\n", len(jobs)-len(failures), len(jobs), strings.Join(failures, "\n"))
	return fmt.Errorf("%d download(s) failed", len(failures))
}

func (cli CLI) downloadJobs() ([]downloadJob, error) {
	if isHTTPURL(cli.Input) {
		output := cli.Output
		if output == "" {
			var err error
			if err := validateOutputDir(cli.OutputDir); err != nil {
				return nil, err
			}

			output, err = deriveOutputPath(cli.Input, cli.OutputDir, nil)
			if err != nil {
				return nil, err
			}
		}

		if err := validateURL(cli.Input); err != nil {
			return nil, err
		}

		output, err := validateOutput(output, cli.Overwrite)
		if err != nil {
			return nil, err
		}

		return []downloadJob{{url: cli.Input, output: output}}, nil
	}

	if cli.Output != "" {
		return nil, errors.New("output path is only valid for single URL downloads")
	}

	return cli.batchJobs(cli.Input)
}

func (cli CLI) batchJobs(path string) ([]downloadJob, error) {
	if err := validateOutputDir(cli.OutputDir); err != nil {
		return nil, err
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("could not open input file: %w", err)
	}
	defer func() { _ = file.Close() }()

	seen := make(map[string]int)
	var jobs []downloadJob
	scanner := bufio.NewScanner(file)
	for line := 1; scanner.Scan(); line++ {
		raw := strings.TrimSpace(scanner.Text())
		if raw == "" || strings.HasPrefix(raw, "#") {
			continue
		}

		url, output, err := parseBatchLine(raw)
		if err != nil {
			return nil, fmt.Errorf("%s:%d: %w", path, line, err)
		}

		if err := validateURL(url); err != nil {
			return nil, fmt.Errorf("%s:%d: %w", path, line, err)
		}

		if output == "" {
			output, err = deriveOutputPath(url, cli.OutputDir, seen)
			if err != nil {
				return nil, fmt.Errorf("%s:%d: %w", path, line, err)
			}
		}

		output, err = validateOutput(output, cli.Overwrite)
		if err != nil {
			return nil, fmt.Errorf("%s:%d: %w", path, line, err)
		}

		jobs = append(jobs, downloadJob{url: url, output: output})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("could not read input file: %w", err)
	}
	if len(jobs) == 0 {
		return nil, errors.New("input file did not contain any URLs")
	}

	return jobs, nil
}

func parseBatchLine(line string) (string, string, error) {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return "", "", errors.New("missing URL")
	}

	url := fields[0]
	output := strings.TrimSpace(strings.TrimPrefix(line, url))
	url = strings.TrimSpace(url)
	output = strings.TrimSpace(output)

	if url == "" {
		return "", "", errors.New("missing URL")
	}

	return url, output, nil
}

func validateURL(raw string) error {
	parsed, err := url.ParseRequestURI(raw)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	switch parsed.Scheme {
	case "http", "https":
		return nil
	default:
		return fmt.Errorf("URL must use http or https, got %q", parsed.Scheme)
	}
}

func validateOutput(output string, overwrite bool) (string, error) {
	if strings.TrimSpace(output) == "" {
		return "", errors.New("output path is required")
	}

	if strings.ToLower(filepath.Ext(output)) != ".mp4" {
		return "", errors.New("output path must end in .mp4")
	}

	if _, err := os.Stat(output); err == nil && !overwrite {
		return "", fmt.Errorf("%s already exists; pass --overwrite to replace it", output)
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("could not inspect output path: %w", err)
	}

	return output, nil
}

func validateOutputDir(outputDir string) error {
	info, err := os.Stat(outputDir)
	if err != nil {
		return fmt.Errorf("could not inspect output directory: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("output directory is not a directory: %s", outputDir)
	}
	return nil
}

func isHTTPURL(raw string) bool {
	parsed, err := url.ParseRequestURI(raw)
	if err != nil {
		return false
	}
	return parsed.Scheme == "http" || parsed.Scheme == "https"
}

func deriveOutputPath(rawURL, outputDir string, seen map[string]int) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}

	name := derivedNameFromPath(parsed.Path)
	if name == "" {
		name = "download"
	}

	name = sanitizeFilename(name)
	if name == "" {
		name = "download"
	}

	base := strings.TrimSuffix(name, filepath.Ext(name))
	if base == "" {
		base = "download"
	}

	filename := base + ".mp4"
	if seen != nil {
		seen[filename]++
		if seen[filename] > 1 {
			filename = fmt.Sprintf("%s-%d.mp4", base, seen[filename])
		}
	}

	return filepath.Join(outputDir, filename), nil
}

func derivedNameFromPath(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	for i := len(parts) - 1; i >= 0; i-- {
		part := strings.TrimSpace(parts[i])
		if part == "" || isGenericPlaylistName(part) {
			continue
		}
		return part
	}
	return ""
}

func isGenericPlaylistName(name string) bool {
	lower := strings.ToLower(name)
	switch lower {
	case "seg.m3u8", "index.m3u8", "master.m3u8", "playlist.m3u8", "manifest.m3u8":
		return true
	default:
		return lower == "seg" || lower == "index" || lower == "master" || lower == "playlist" || lower == "manifest"
	}
}

func sanitizeFilename(name string) string {
	var builder strings.Builder
	for _, char := range name {
		switch {
		case char >= 'a' && char <= 'z':
			builder.WriteRune(char)
		case char >= 'A' && char <= 'Z':
			builder.WriteRune(char)
		case char >= '0' && char <= '9':
			builder.WriteRune(char)
		case char == '.', char == '-', char == '_', char == ' ':
			builder.WriteRune(char)
		default:
			builder.WriteRune('-')
		}
	}

	return strings.Trim(builder.String(), ".- _")
}
