# Agents Development Guide

## Build/Test Commands
- `make build` - Build binary to `tmp/symbolista`. ALWAYS use this one after you're done with your changes.
- `make test` - Run all tests
- `make lint` - Run `go vet` and `go fmt`
- `go test ./internal/counter` - Run specific package tests
- `go test -run TestSpecificFunction` - Run specific test

## Code Style
- Use standard Go formatting with `go fmt`
- Use structured logging with `slog` via `internal/logger`
- Error handling: wrap errors with `fmt.Errorf("context: %w", err)`
- Use receiver methods for types (e.g. `func (c CharCounts) Len()`)
- Comments are not allowed by default. Only when the code is very difficult to understand without, and in that case you should probably refactor it to be simpler.

## Project Structure
- CLI commands in `cmd/` using Cobra framework
- Business logic in `internal/` packages
- Tests alongside code files with `_test.go` suffix
- Use dependency injection pattern for components
- TUI mode with bubbletea and ntcharts.
