package logger

import (
	"log/slog"
	"os"
)

var (
	defaultLogger *slog.Logger
	verboseCount  int
)

func init() {
	defaultLogger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))
	slog.SetDefault(defaultLogger)
}

func SetVerbosity(count int) {
	verboseCount = count
	var level slog.Level

	switch count {
	case 0:
		level = slog.LevelError
	case 1:
		level = slog.LevelInfo
	case 2:
		level = slog.LevelDebug
	case 3:
		level = slog.LevelDebug - 1 // Extra verbose
	default:
		level = slog.LevelDebug - 1
	}

	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	})
	defaultLogger = slog.New(handler)
	slog.SetDefault(defaultLogger)
}

func GetVerbosity() int {
	return verboseCount
}

func Info(msg string, args ...any) {
	defaultLogger.Info(msg, args...)
}

func Debug(msg string, args ...any) {
	defaultLogger.Debug(msg, args...)
}

func Trace(msg string, args ...any) {
	defaultLogger.Log(nil, slog.LevelDebug-1, msg, args...)
}

func Error(msg string, args ...any) {
	defaultLogger.Error(msg, args...)
}
