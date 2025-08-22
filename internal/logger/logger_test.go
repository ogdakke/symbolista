package logger

import (
	"testing"
)

func TestSetVerbosity(t *testing.T) {
	originalVerbosity := GetVerbosity()
	defer SetVerbosity(originalVerbosity)

	tests := []int{0, 1, 2, 3, 4}

	for _, verbosity := range tests {
		SetVerbosity(verbosity)

		if GetVerbosity() != verbosity {
			t.Errorf("Expected verbosity %d, got %d", verbosity, GetVerbosity())
		}

		Error("error message")
		Info("info message")
		Debug("debug message")
		Trace("trace message")
	}
}

func TestGetVerbosity(t *testing.T) {
	originalVerbosity := GetVerbosity()
	defer SetVerbosity(originalVerbosity)

	SetVerbosity(2)
	if GetVerbosity() != 2 {
		t.Errorf("Expected verbosity 2, got %d", GetVerbosity())
	}

	SetVerbosity(0)
	if GetVerbosity() != 0 {
		t.Errorf("Expected verbosity 0, got %d", GetVerbosity())
	}
}

func TestLoggerFunctions(t *testing.T) {
	originalVerbosity := GetVerbosity()
	defer SetVerbosity(originalVerbosity)

	SetVerbosity(3)

	Info("test info", "key", "value")
	Debug("test debug", "number", 42)
	Trace("test trace", "bool", true)
	Error("test error", "error", "something went wrong")
}

func TestLoggerWithNoArgs(t *testing.T) {
	originalVerbosity := GetVerbosity()
	defer SetVerbosity(originalVerbosity)

	SetVerbosity(3)

	Info("simple info")
	Debug("simple debug")
	Trace("simple trace")
	Error("simple error")
}

func TestDefaultLogger(t *testing.T) {
	if defaultLogger == nil {
		t.Error("Default logger should be initialized")
	}

	if verboseCount != 0 {
		t.Errorf("Initial verbosity should be 0, got %d", verboseCount)
	}
}

func TestVerbosityBounds(t *testing.T) {
	originalVerbosity := GetVerbosity()
	defer SetVerbosity(originalVerbosity)

	SetVerbosity(-1)
	if GetVerbosity() != -1 {
		t.Errorf("Expected verbosity -1, got %d", GetVerbosity())
	}

	SetVerbosity(10)
	if GetVerbosity() != 10 {
		t.Errorf("Expected verbosity 10, got %d", GetVerbosity())
	}

	Error("error at extreme verbosity")
	Info("info at extreme verbosity")
	Debug("debug at extreme verbosity")
	Trace("trace at extreme verbosity")
}
