package counter

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCharCountSorting(t *testing.T) {
	counts := CharCounts{
		{Char: 'a', Count: 5, Percentage: 50.0},
		{Char: 'b', Count: 3, Percentage: 30.0},
		{Char: 'c', Count: 2, Percentage: 20.0},
	}

	// They should already be sorted by count (descending)
	if counts[0].Count != 5 || counts[1].Count != 3 || counts[2].Count != 2 {
		t.Error("CharCounts should be sorted by count in descending order")
	}

	// Test the Less method
	// Less(i, j) returns true if c[i].Count > c[j].Count (descending order)
	if counts.Less(0, 1) != true { // 5 > 3, so true
		t.Error("Less method should return true when first count is greater")
	}
	if counts.Less(1, 0) != false { // 3 < 5, so false
		t.Error("Less method should return false when first count is smaller")
	}

	// Test the Len method
	if counts.Len() != 3 {
		t.Errorf("Expected length 3, got %d", counts.Len())
	}

	// Test the Swap method
	counts.Swap(0, 2)
	if counts[0].Char != 'c' || counts[2].Char != 'a' {
		t.Error("Swap method did not work correctly")
	}
}

func TestOutputJSON(t *testing.T) {
	counts := CharCounts{
		{Char: 'a', Count: 5, Percentage: 50.0},
		{Char: 'b', Count: 3, Percentage: 30.0},
	}

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	outputJSON(counts, true)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Verify it's valid JSON
	var result []CharCount
	err := json.Unmarshal([]byte(output), &result)
	if err != nil {
		t.Fatalf("Output is not valid JSON: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("Expected 2 items in JSON output, got %d", len(result))
	}

	if result[0].Char != 'a' || result[0].Count != 5 || result[0].Percentage != 50.0 {
		t.Errorf("First JSON item incorrect: %+v", result[0])
	}
}

func TestOutputJSONWithoutPercentages(t *testing.T) {
	counts := CharCounts{
		{Char: 'a', Count: 5, Percentage: 50.0},
	}

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	outputJSON(counts, false)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Verify JSON and that percentages are 0
	var result []CharCount
	err := json.Unmarshal([]byte(output), &result)
	if err != nil {
		t.Fatalf("Output is not valid JSON: %v", err)
	}

	if result[0].Percentage != 0 {
		t.Errorf("Expected percentage to be 0 when showPercentages is false, got %f", result[0].Percentage)
	}
}

func TestOutputCSV(t *testing.T) {
	counts := CharCounts{
		{Char: 'a', Count: 5, Percentage: 50.0},
		{Char: ' ', Count: 3, Percentage: 30.0},
		{Char: '\n', Count: 2, Percentage: 20.0},
	}

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	outputCSV(counts, true)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 4 { // header + 3 data rows
		t.Errorf("Expected 4 lines in CSV output, got %d", len(lines))
	}

	// Check header
	if !strings.Contains(lines[0], "Character") || !strings.Contains(lines[0], "Count") || !strings.Contains(lines[0], "Percentage") {
		t.Errorf("CSV header incorrect: %s", lines[0])
	}

	// Check special character formatting
	if !strings.Contains(lines[2], "<space>") {
		t.Error("Space character should be formatted as <space>")
	}
	if !strings.Contains(lines[3], "<newline>") {
		t.Error("Newline character should be formatted as <newline>")
	}
}

func TestOutputTable(t *testing.T) {
	counts := CharCounts{
		{Char: 'a', Count: 5, Percentage: 50.0},
		{Char: '\t', Count: 3, Percentage: 30.0},
		{Char: '\r', Count: 2, Percentage: 20.0},
	}

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	outputTable(counts, true)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Check that special characters are formatted correctly
	if !strings.Contains(output, "<tab>") {
		t.Error("Tab character should be formatted as <tab>")
	}
	if !strings.Contains(output, "<return>") {
		t.Error("Return character should be formatted as <return>")
	}
	if !strings.Contains(output, "Character") {
		t.Error("Table should have Character header")
	}
	if !strings.Contains(output, "Count") {
		t.Error("Table should have Count header")
	}
	if !strings.Contains(output, "Percentage") {
		t.Error("Table should have Percentage header when showPercentages is true")
	}
}

func TestOutputTableWithoutPercentages(t *testing.T) {
	counts := CharCounts{
		{Char: 'a', Count: 5, Percentage: 50.0},
	}

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	outputTable(counts, false)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if strings.Contains(output, "Percentage") {
		t.Error("Table should not have Percentage header when showPercentages is false")
	}
}

func TestCountSymbolsIntegration(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "counter_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files
	testFile1 := filepath.Join(tempDir, "test1.txt")
	err = os.WriteFile(testFile1, []byte("aaa"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	testFile2 := filepath.Join(tempDir, "test2.txt")
	err = os.WriteFile(testFile2, []byte("bb"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create .gitignore to ignore one file
	gitignoreFile := filepath.Join(tempDir, ".gitignore")
	err = os.WriteFile(gitignoreFile, []byte("test2.txt\n"), 0644)
	if err != nil {
		t.Fatalf("Failed to create .gitignore: %v", err)
	}

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run CountSymbols with JSON output
	CountSymbols(tempDir, "json", true)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Parse JSON output
	var result []CharCount
	err = json.Unmarshal([]byte(output), &result)
	if err != nil {
		t.Fatalf("CountSymbols output is not valid JSON: %v", err)
	}

	// Should count characters from test1.txt (aaa) and .gitignore file (test2.txt\n)
	// The .gitignore file contains "test2.txt\n" which adds more characters
	// Let's check that 'a' is present with count 3
	found_a := false
	for _, char := range result {
		if char.Char == 'a' && char.Count == 3 {
			found_a = true
			break
		}
	}

	if !found_a {
		t.Error("Expected to find character 'a' with count 3")
	}
}

func TestCountSymbolsWithMultipleFormats(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "counter_format_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a simple test file
	testFile := filepath.Join(tempDir, "test.txt")
	err = os.WriteFile(testFile, []byte("ab"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	formats := []string{"json", "csv", "table"}

	for _, format := range formats {
		t.Run("format_"+format, func(t *testing.T) {
			// Capture stdout
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			CountSymbols(tempDir, format, false)

			w.Close()
			os.Stdout = old

			var buf bytes.Buffer
			io.Copy(&buf, r)
			output := buf.String()

			if output == "" {
				t.Errorf("No output generated for format %s", format)
			}

			// Basic validation based on format
			switch format {
			case "json":
				var result []CharCount
				err := json.Unmarshal([]byte(output), &result)
				if err != nil {
					t.Errorf("Invalid JSON output for format %s: %v", format, err)
				}
			case "csv":
				if !strings.Contains(output, "Character,Count") {
					t.Errorf("CSV output missing expected header for format %s", format)
				}
			case "table":
				if !strings.Contains(output, "Character") || !strings.Contains(output, "Count") {
					t.Errorf("Table output missing expected headers for format %s", format)
				}
			}
		})
	}
}

func TestCountSymbolsNonExistentDirectory(t *testing.T) {
	// Capture stdout and stderr
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stdout = w
	os.Stderr = w

	CountSymbols("/nonexistent/directory", "json", true)

	w.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Should contain some error message or warning
	if !strings.Contains(output, "Error") && !strings.Contains(output, "error") {
		t.Error("Expected error message for nonexistent directory")
	}
}
