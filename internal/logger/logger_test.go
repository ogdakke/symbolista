package logger

import (
	"testing"
)

func TestSetVerbosity(t *testing.T) {
	originalVerbosity := GetVerbosity()
	defer SetVerbosity(originalVerbosity) // Restore original verbosity

	tests := []int{0, 1, 2, 3, 4}

	for _, verbosity := range tests {
		SetVerbosity(verbosity)

		if GetVerbosity() != verbosity {
			t.Errorf("Expected verbosity %d, got %d", verbosity, GetVerbosity())
		}

		// Test that the logger functions don't panic at this verbosity level
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

	// Set to max verbosity to enable all log levels
	SetVerbosity(3)

	// Test that all logger functions work without panicking
	Info("test info", "key", "value")
	Debug("test debug", "number", 42)
	Trace("test trace", "bool", true)
	Error("test error", "error", "something went wrong")
}

func TestLoggerWithNoArgs(t *testing.T) {
	originalVerbosity := GetVerbosity()
	defer SetVerbosity(originalVerbosity)

	SetVerbosity(3)

	// Test logger functions with no additional args
	Info("simple info")
	Debug("simple debug")
	Trace("simple trace")
	Error("simple error")
}

func TestDefaultLogger(t *testing.T) {
	// Test that the default logger is initialized correctly
	if defaultLogger == nil {
		t.Error("Default logger should be initialized")
	}

	// Test that the initial verbosity is 0
	if verboseCount != 0 {
		t.Errorf("Initial verbosity should be 0, got %d", verboseCount)
	}
}

func TestVerbosityBounds(t *testing.T) {
	originalVerbosity := GetVerbosity()
	defer SetVerbosity(originalVerbosity)

	// Test setting negative verbosity (should work but cap at 0 behavior)
	SetVerbosity(-1)
	if GetVerbosity() != -1 {
		t.Errorf("Expected verbosity -1, got %d", GetVerbosity())
	}

	// Test setting very high verbosity (should work but cap at max behavior)
	SetVerbosity(10)
	if GetVerbosity() != 10 {
		t.Errorf("Expected verbosity 10, got %d", GetVerbosity())
	}

	// Test that logger functions still work at extreme values
	Error("error at extreme verbosity")
	Info("info at extreme verbosity")
	Debug("debug at extreme verbosity")
	Trace("trace at extreme verbosity")
}
