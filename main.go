package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/alecthomas/kong"
)

type CLI struct {
	Input     string        `arg:"" name:"input" help:"HTTP(S) m3u8 playlist URL or text file with one URL and optional output path per line."`
	Output    string        `arg:"" optional:"" name:"output" help:"Local .mp4 output path for single URL downloads." type:"path"`
	OutputDir string        `help:"Directory for derived output filenames." type:"path" default:"."`
	Overwrite bool          `short:"f" help:"Overwrite the output file if it already exists."`
	Quiet     bool          `short:"q" help:"Reduce ffmpeg console output."`
	Verbose   bool          `short:"v" help:"Show raw ffmpeg output for debugging."`
	Timeout   time.Duration `help:"Optional overall timeout, for example 30m or 1h."`
}

func main() {
	var cli CLI
	ctx := kong.Parse(&cli,
		kong.Name("hlsdl"),
		kong.Description("Download an HLS stream from an m3u8 manifest to a local mp4 file using ffmpeg."),
		kong.UsageOnError(),
	)

	if err := cli.Run(context.Background(), os.Stdout, os.Stderr); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error: %v\n", err)
		ctx.Exit(1)
	}
}
