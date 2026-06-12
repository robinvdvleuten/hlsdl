# hlsdl

A small Go CLI for downloading HLS (`.m3u8`) streams to local MP4 files.

It uses `ffmpeg` for the actual download/remux work, `ffprobe` to select the
best available stream when possible, and a compact terminal progress indicator
while the download runs.

## Features

- Download a single `.m3u8` URL to MP4.
- Download multiple URLs from a text file.
- Optional output filenames for both single and batch downloads.
- Automatic MP4 filename derivation when no output path is provided.
- Best-effort highest-quality stream selection via `ffprobe`.
- Per-download progress bar when duration is known, with spinner fallback.
- Continues batch downloads after individual failures.

## Requirements

- Go 1.24 or newer
- `ffmpeg`
- `ffprobe`

On macOS with Homebrew:

```sh
brew install ffmpeg
```

## Install

From this repository:

```sh
go install .
```

Or build a local binary:

```sh
go build -o hlsdl .
```

## Usage

```sh
hlsdl <input> [output] [flags]
```

`input` can be either:

- an HTTP(S) `.m3u8` URL
- a text file containing multiple downloads

## Changelog

Please see [CHANGELOG](CHANGELOG.md) for more information on what has changed recently.

### Single download

Let the tool derive the output filename:

```sh
hlsdl "https://example.com/video/SEG.m3u8"
```

Write to a specific file:

```sh
hlsdl "https://example.com/video/SEG.m3u8" episode-01.mp4
```

Write derived filenames into another directory:

```sh
hlsdl "https://example.com/video/SEG.m3u8" --output-dir videos
```

### Batch downloads

Pass a text file as the first argument:

```sh
hlsdl downloads.txt
```

With an output directory:

```sh
hlsdl downloads.txt --output-dir videos
```

Each batch item is downloaded sequentially. The CLI prints an item indicator
before each download:

```text
[1/4] episode-01.mp4
[2/4] episode-02.mp4
```

There is no aggregate progress bar for the whole batch; each download keeps its
own progress indicator.

## Batch File Format

Each non-empty line contains a URL and, optionally, an output path:

```txt
# URL only: output filename is derived
https://example.com/show/episode-01/SEG.m3u8

# URL plus explicit output path
https://example.com/show/episode-02/SEG.m3u8 episode-02.mp4

# Output paths may contain spaces
https://example.com/show/episode-03/SEG.m3u8 My Episode 03.mp4
```

Rules:

- Empty lines are ignored.
- Lines starting with `#` are ignored.
- The first field is always the URL.
- Everything after the first whitespace separator is treated as the output path.
- URL-only lines derive an `.mp4` filename and use `--output-dir`.

## Flags

```text
--output-dir string   Directory for derived output filenames (default ".")
-f, --overwrite       Overwrite the output file if it already exists
-q, --quiet           Reduce console output
-v, --verbose         Show raw ffmpeg output for debugging
--timeout duration    Optional overall timeout, for example 30m or 1h
```

## Output Names

When no output path is provided, the filename is derived from the URL path.
Generic playlist names such as `SEG.m3u8`, `index.m3u8`, `master.m3u8`, and
`playlist.m3u8` fall back to the parent path segment.

Examples:

```text
https://example.com/video/SEG.m3u8      -> video.mp4
https://example.com/show/episode.m3u8   -> episode.mp4
```

In batch mode, duplicate derived names are numbered:

```text
episode.mp4
episode-2.mp4
episode-3.mp4
```

Existing files are not overwritten unless `--overwrite` is passed.

## Notes

For master playlists, the tool asks `ffprobe` for available streams and maps the
best video stream it can identify, preferring higher bitrate and then higher
resolution. Audio from the same program is used when available.

If probing fails or duration is unavailable, the download still proceeds with
ffmpeg defaults and the UI falls back to an indeterminate spinner.

DRM-protected streams are not supported.

## Contributing

Everyone is encouraged to help improve this project. Here are a few ways you can help:

- [Report bugs](https://github.com/robinvdvleuten/hlsdl/issues)
- Fix bugs and [submit pull requests](https://github.com/robinvdvleuten/hlsdl/pulls)
- Write, clarify, or fix documentation
- Suggest or add new features

To get started with development:

```
git clone https://github.com/robinvdvleuten/hlsdl.git
cd hlsdl
go test ./...
```

Before submitting a pull request, please make sure to run `go fmt` on any Go source files you touched so the code stays consistent.

Feel free to open an issue to get feedback on your idea before spending too much time on it.

## License

The MIT License (MIT). Please see [License File](LICENSE.md) for more information.
