package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func (cli CLI) runDownload(ctx context.Context, cmd *exec.Cmd, stdout, stderr io.Writer, total time.Duration) error {
	if cli.Verbose {
		cmd.Stdout = stdout
		cmd.Stderr = stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("ffmpeg failed: %w", err)
		}
		return nil
	}

	progress, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("could not read ffmpeg progress: %w", err)
	}

	var ffmpegErr bytes.Buffer
	cmd.Stderr = &ffmpegErr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("could not start ffmpeg: %w", err)
	}

	var program *tea.Program
	programDone := make(chan error, 1)
	if !cli.Quiet {
		program = tea.NewProgram(newProgressModel(total), tea.WithOutput(stdout), tea.WithInput(nil))
		go func() {
			_, err := program.Run()
			programDone <- err
		}()
	}

	scannerErr := scanProgress(progress, func(update progressUpdateMsg) {
		if program != nil {
			program.Send(update)
		}
	})

	runErr := cmd.Wait()
	if program != nil {
		program.Send(downloadDoneMsg{})
		if err := <-programDone; err != nil && runErr == nil {
			runErr = err
		}
	}

	if scannerErr != nil && runErr == nil {
		runErr = scannerErr
	}
	if runErr != nil {
		message := strings.TrimSpace(ffmpegErr.String())
		if message != "" {
			return fmt.Errorf("ffmpeg failed: %w\n%s", runErr, message)
		}
		return fmt.Errorf("ffmpeg failed: %w", runErr)
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}

	return nil
}

func ffmpegArgs(inputURL, output string, overwrite, verbose bool, mappings []int) []string {
	args := []string{"-hide_banner"}

	if overwrite {
		args = append(args, "-y")
	} else {
		args = append(args, "-n")
	}

	if !verbose {
		args = append(args, "-loglevel", "error", "-progress", "pipe:1", "-stats_period", "1")
	}

	args = append(args, "-i", inputURL)

	for _, mapping := range mappings {
		args = append(args, "-map", fmt.Sprintf("0:%d", mapping))
	}

	args = append(args,
		"-c", "copy",
		"-bsf:a", "aac_adtstoasc",
		output,
	)

	return args
}
