# Symbol Counter

this counts the occurrences of symbols in a directory's files.

I used it to decide which keys I wanted to remap, based on the frequency of use while programming

## Installation

```sh
brew install symbolista
```

or use go

```sh
go install github.com/ogdakke/symbolista@latest
```

## Usage

```sh
symbolista [directory] [flags]

Flags:
    --ascii-only         Count only ASCII characters (0-127). Use --ascii-only=false to include all Unicode characters (default true)
-f, --format string      Output format (table, json, csv) (default "table")
-h, --help               help for symbolista
    --include-dotfiles   Include dotfiles in analysis (by default dotfiles are ignored)
-p, --percentages        Show percentages in output (default true)
    --tui                Launch interactive TUI interface
-V, --verbose count      Increase verbosity (-V info, -VV debug, -VVV trace)
-v, --version            Show version and exit
-w, --workers int        Number of worker goroutines (0 = auto-detect based on CPU cores) (default 0)
```

## Development

Use make to build and lint and test the program.
