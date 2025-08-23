package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ogdakke/symbolista/internal/domain"
)

func TestExecuteWithDefaultArgs(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cmd_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testFile := filepath.Join(tempDir, "test.txt")
	err = os.WriteFile(testFile, []byte("abc"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	originalFormat := outputFormat
	originalPercentages := showPercentages
	originalVerbosity := verboseCount
	originalArgs := os.Args

	outputFormat = "json"
	showPercentages = true
	verboseCount = 0

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	os.Args = []string{"symbolista"}
	rootCmd.Run(rootCmd, []string{tempDir})

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	outputFormat = originalFormat
	showPercentages = originalPercentages
	verboseCount = originalVerbosity
	os.Args = originalArgs

	var jsonOutput domain.JSONOutput
	err = json.Unmarshal([]byte(output), &jsonOutput)
	if err != nil {
		t.Fatalf("Command output is not valid JSON: %v", err)
	}
	result := jsonOutput.Result

	if len(result.Characters) != 3 {
		t.Errorf("Expected 3 characters, got %d", len(result.Characters))
	}

	for _, char := range result.Characters {
		if char.Count != 1 {
			t.Errorf("Expected count 1 for character %s, got %d", char.Char, char.Count)
		}
		if char.Percentage < 33.0 || char.Percentage > 34.0 {
			t.Errorf("Expected percentage around 33.33 for character %s, got %f", char.Char, char.Percentage)
		}
	}
}

func TestExecuteWithDirectoryArg(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cmd_arg_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testFile := filepath.Join(tempDir, "test.txt")
	err = os.WriteFile(testFile, []byte("xyz"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	originalFormat := outputFormat
	originalPercentages := showPercentages
	originalVerbosity := verboseCount

	outputFormat = "json"
	showPercentages = false
	verboseCount = 0

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	rootCmd.Run(rootCmd, []string{tempDir})

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	outputFormat = originalFormat
	showPercentages = originalPercentages
	verboseCount = originalVerbosity

	var jsonOutput domain.JSONOutput
	err = json.Unmarshal([]byte(output), &jsonOutput)
	if err != nil {
		t.Fatalf("Command output is not valid JSON: %v", err)
	}
	result := jsonOutput.Result

	if len(result.Characters) != 3 {
		t.Errorf("Expected 3 characters, got %d", len(result.Characters))
	}

	for _, char := range result.Characters {
		if char.Percentage != 0 {
			t.Errorf("Expected percentage 0 when showPercentages is false, got %f", char.Percentage)
		}
	}
}

func TestExecuteWithCSVFormat(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "cmd_csv_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test file
	testFile := filepath.Join(tempDir, "test.txt")
	err = os.WriteFile(testFile, []byte("a,b"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Save original values
	originalFormat := outputFormat
	originalPercentages := showPercentages
	originalVerbosity := verboseCount

	// Set test values
	outputFormat = "csv"
	showPercentages = true
	verboseCount = 0

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Execute the command
	rootCmd.Run(rootCmd, []string{tempDir})

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Restore original values
	outputFormat = originalFormat
	showPercentages = originalPercentages
	verboseCount = originalVerbosity

	// Verify CSV format
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) < 2 {
		t.Errorf("Expected at least 2 lines in CSV output, got %d", len(lines))
	}

	if !strings.Contains(lines[0], "type") || !strings.Contains(lines[0], "sequence") || !strings.Contains(lines[0], "count") {
		t.Errorf("CSV header incorrect: %s", lines[0])
	}
}

func TestExecuteWithTableFormat(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "cmd_table_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test file with special characters
	testFile := filepath.Join(tempDir, "test.txt")
	err = os.WriteFile(testFile, []byte("a\nb\tc"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Save original values
	originalFormat := outputFormat
	originalPercentages := showPercentages
	originalVerbosity := verboseCount

	// Set test values
	outputFormat = "table"
	showPercentages = true
	verboseCount = 0

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Execute the command
	rootCmd.Run(rootCmd, []string{tempDir})

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Restore original values
	outputFormat = originalFormat
	showPercentages = originalPercentages
	verboseCount = originalVerbosity

	// Verify table format
	if !strings.Contains(output, "Character") || !strings.Contains(output, "Count") {
		t.Error("Table output missing expected headers")
	}

	if !strings.Contains(output, "<newline>") {
		t.Error("Newline should be formatted as <newline>")
	}
	if !strings.Contains(output, "<tab>") {
		t.Error("Tab should be formatted as <tab>")
	}
}

func TestExecuteWithGitignore(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "cmd_gitignore_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files
	includeFile := filepath.Join(tempDir, "include.txt")
	err = os.WriteFile(includeFile, []byte("include"), 0644)
	if err != nil {
		t.Fatalf("Failed to create include file: %v", err)
	}

	ignoreFile := filepath.Join(tempDir, "ignore.log")
	err = os.WriteFile(ignoreFile, []byte("ignore"), 0644)
	if err != nil {
		t.Fatalf("Failed to create ignore file: %v", err)
	}

	// Create .gitignore
	gitignoreFile := filepath.Join(tempDir, ".gitignore")
	err = os.WriteFile(gitignoreFile, []byte("*.log\n"), 0644)
	if err != nil {
		t.Fatalf("Failed to create .gitignore: %v", err)
	}

	// Save original values
	originalFormat := outputFormat
	originalPercentages := showPercentages
	originalVerbosity := verboseCount

	// Set test values
	outputFormat = "json"
	showPercentages = false
	verboseCount = 0

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Execute the command
	rootCmd.Run(rootCmd, []string{tempDir})

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Restore original values
	outputFormat = originalFormat
	showPercentages = originalPercentages
	verboseCount = originalVerbosity

	// Verify JSON output
	var jsonOutput domain.JSONOutput
	err = json.Unmarshal([]byte(output), &jsonOutput)
	if err != nil {
		t.Fatalf("Command output is not valid JSON: %v", err)
	}
	result := jsonOutput.Result

	// Should include characters from "include" and from ".gitignore" file content ("*.log\n")
	// Should not include characters from "ignore.log" file content
	expectedChars := map[rune]bool{'i': true, 'n': true, 'c': true, 'l': true, 'u': true, 'd': true, 'e': true}
	foundChars := make(map[rune]bool)

	for _, char := range result.Characters {
		if len(char.Char) > 0 {
			foundChars[rune(char.Char[0])] = true
		}
	}

	for expectedChar := range expectedChars {
		if !foundChars[expectedChar] {
			t.Errorf("Expected character %c not found in output", expectedChar)
		}
	}
}

func TestExecuteWithVerbosity(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "cmd_verbose_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test file
	testFile := filepath.Join(tempDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Save original values
	originalFormat := outputFormat
	originalPercentages := showPercentages
	originalVerbosity := verboseCount

	// Set test values
	outputFormat = "json"
	showPercentages = false
	verboseCount = 2 // Debug level

	// Capture stdout and stderr
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	os.Stdout = wOut
	os.Stderr = wErr

	// Execute the command
	rootCmd.Run(rootCmd, []string{tempDir})

	wOut.Close()
	wErr.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	var bufOut, bufErr bytes.Buffer
	bufOut.ReadFrom(rOut)
	bufErr.ReadFrom(rErr)
	stdout := bufOut.String()
	stderr := bufErr.String()

	// Restore original values
	outputFormat = originalFormat
	showPercentages = originalPercentages
	verboseCount = originalVerbosity

	// Verify JSON output still works
	var jsonOutput domain.JSONOutput
	err = json.Unmarshal([]byte(stdout), &jsonOutput)
	if err != nil {
		t.Fatalf("Command output is not valid JSON: %v", err)
	}

	if !strings.Contains(stderr, "Starting symbol analysis") {
		t.Error("Expected debug log message not found in stderr")
	}
}

func TestExecuteWithNonExistentDirectory(t *testing.T) {
	// Save original values
	originalFormat := outputFormat
	originalVerbosity := verboseCount

	// Set test values
	outputFormat = "json"
	verboseCount = 0

	// Capture stdout and stderr
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	os.Stdout = wOut
	os.Stderr = wErr

	// Execute the command
	rootCmd.Run(rootCmd, []string{"/nonexistent/directory"})

	wOut.Close()
	wErr.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	var bufOut, bufErr bytes.Buffer
	bufOut.ReadFrom(rOut)
	bufErr.ReadFrom(rErr)
	stdout := bufOut.String()
	stderr := bufErr.String()

	// Restore original values
	outputFormat = originalFormat
	verboseCount = originalVerbosity

	combined := stdout + stderr
	if !strings.Contains(combined, "Error") && !strings.Contains(combined, "error") {
		t.Error("Expected error message for nonexistent directory")
	}
}
