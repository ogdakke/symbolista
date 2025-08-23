# Symbol Counter

this counts the occurrences of symbols in a directory's files.

I used it to decide which keys I wanted to remap, based on the frequency of use while programming

## Installation

```sh
brew install --cask ogdakke/homebrew-tap/symbolista
```

or use go

```sh
go install github.com/ogdakke/symbolista@latest
```

## Usage

```sh
Usage:
  symbolista [directory] [flags]

Flags:
      --ascii-only         Count only ASCII characters. Use --ascii-only=false to include all Unicode characters (default true)
  -f, --format string      Output format (table, json, csv) (default "table")
  -j, --from-json string   Load data from JSON file and launch TUI (requires --tui flag)
  -h, --help               help for symbolista
      --include-dotfiles   Include dotfiles in analysis (default false)
  -m, --metadata           Include metadata in JSON output (directory, file counts, timing info) (default true) (default true)
  -p, --percentages        Show percentages in output (default true)
      --tui                Launch interactive TUI interface
  -V, --verbose count      Increase verbosity (-V info, -VV debug, -VVV trace)
  -v, --version            Show version and exit
  -w, --workers int        Number of worker goroutines (0 = auto-detect based on CPU cores) (default 0)
```

## Examples

See [examples](./examples/) for some example outputs from known repositories, namely linux and vscode.

## Development

Use make to build and lint and test the program.
