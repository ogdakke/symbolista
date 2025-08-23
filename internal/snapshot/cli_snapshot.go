package snapshot

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sergi/go-diff/diffmatchpatch"
)

// CLISnapshot represents the complete output of running the CLI tool
type CLISnapshot struct {
	TestName    string    `json:"test_name"`
	Directory   string    `json:"directory"`
	Args        []string  `json:"args"`
	ExitCode    int       `json:"exit_code"`
	StdoutLines []string  `json:"stdout_lines"`
	StderrLines []string  `json:"stderr_lines"`
	JSONOutput  *JSONData `json:"json_output,omitempty"` // Only for JSON format tests
}

// JSONData represents the JSON output structure for validation
type JSONData struct {
	Result   JSONResult    `json:"result"`
	Metadata *JSONMetadata `json:"metadata,omitempty"`
}

type JSONResult struct {
	Characters []CharCount     `json:"characters"`
	Sequences  []SequenceCount `json:"sequences"`
}

type CharCount struct {
	Char       string  `json:"char"`
	Count      int     `json:"count"`
	Percentage float64 `json:"percentage"`
}

type SequenceCount struct {
	Sequence   string  `json:"sequence"`
	Count      int     `json:"count"`
	Percentage float64 `json:"percentage"`
}

type JSONMetadata struct {
	Directory       string `json:"directory"`
	FilesFound      int    `json:"files_found"`
	FilesProcessed  int    `json:"files_processed"`
	FilesIgnored    int    `json:"files_ignored"`
	TotalCharacters int    `json:"total_characters"`
	UniqueChars     int    `json:"unique_characters"`
}

type CLISnapshotTester struct {
	snapshotDir  string
	baselineMode bool
	binaryPath   string
}

func NewCLISnapshotTester(snapshotDir string, baselineMode bool, binaryPath string) *CLISnapshotTester {
	return &CLISnapshotTester{
		snapshotDir:  snapshotDir,
		baselineMode: baselineMode,
		binaryPath:   binaryPath,
	}
}

func (st *CLISnapshotTester) Test(t *testing.T, testName string, testDir string, args []string) {
	t.Helper()

	// Build full command args
	fullArgs := append(args, testDir)

	// Run the CLI command
	snapshot, err := st.runCLI(testName, testDir, fullArgs)
	if err != nil {
		t.Fatalf("Failed to run CLI command: %v", err)
	}

	snapshotPath := filepath.Join(st.snapshotDir, fmt.Sprintf("%s.json", testName))

	if st.baselineMode {
		// Create/update the snapshot
		st.createSnapshot(t, *snapshot, snapshotPath)
		t.Logf("Created snapshot: %s", snapshotPath)
	} else {
		// Compare with existing snapshot
		st.compareSnapshot(t, *snapshot, snapshotPath)
	}
}

func (st *CLISnapshotTester) runCLI(testName, testDir string, args []string) (*CLISnapshot, error) {
	cmd := exec.Command(st.binaryPath, args...)

	// Capture both stdout and stderr
	stdout, err := cmd.Output()
	var stderr []byte
	if exitError, ok := err.(*exec.ExitError); ok {
		stderr = exitError.Stderr
	}

	// Split output into lines for easier comparison
	stdoutLines := strings.Split(strings.TrimSpace(string(stdout)), "\n")
	stderrLines := strings.Split(strings.TrimSpace(string(stderr)), "\n")

	// Remove empty lines from stderr (common in test output)
	stderrLines = filterEmptyLines(stderrLines)

	snapshot := &CLISnapshot{
		TestName:    testName,
		Directory:   testDir,
		Args:        args,
		ExitCode:    cmd.ProcessState.ExitCode(),
		StdoutLines: stdoutLines,
		StderrLines: stderrLines,
	}

	// If this is a JSON format test, parse the JSON for validation
	if contains(args, "--format=json") || contains(args, "-f") && contains(args, "json") {
		var jsonData JSONData
		if err := json.Unmarshal(stdout, &jsonData); err == nil {
			snapshot.JSONOutput = &jsonData
		}
	}

	return snapshot, nil
}

func (st *CLISnapshotTester) createSnapshot(t *testing.T, snapshot CLISnapshot, snapshotPath string) {
	t.Helper()

	// Ensure snapshot directory exists
	if err := os.MkdirAll(st.snapshotDir, 0755); err != nil {
		t.Fatalf("Failed to create snapshot directory: %v", err)
	}

	// Normalize the snapshot for consistent comparison
	normalizedSnapshot := st.normalizeSnapshot(snapshot)

	data, err := json.MarshalIndent(normalizedSnapshot, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal snapshot: %v", err)
	}

	if err := os.WriteFile(snapshotPath, data, 0644); err != nil {
		t.Fatalf("Failed to write snapshot file: %v", err)
	}
}

func (st *CLISnapshotTester) compareSnapshot(t *testing.T, actual CLISnapshot, snapshotPath string) {
	t.Helper()

	data, err := os.ReadFile(snapshotPath)
	if err != nil {
		t.Fatalf("Failed to read snapshot file %s: %v", snapshotPath, err)
	}

	var expected CLISnapshot
	if err := json.Unmarshal(data, &expected); err != nil {
		t.Fatalf("Failed to unmarshal snapshot: %v", err)
	}

	// Normalize both snapshots
	normalizedActual := st.normalizeSnapshot(actual)
	normalizedExpected := st.normalizeSnapshot(expected)

	// Compare normalized snapshots
	if !st.snapshotsEqual(normalizedActual, normalizedExpected) {
		actualData, _ := json.MarshalIndent(normalizedActual, "", "  ")
		expectedData, _ := json.MarshalIndent(normalizedExpected, "", "  ")

		diffOutput := st.generateColoredDiff(string(expectedData), string(actualData))
		t.Errorf("Snapshot mismatch for %s\n%s", actual.TestName, diffOutput)
	}
}

func (st *CLISnapshotTester) normalizeSnapshot(snapshot CLISnapshot) CLISnapshot {
	// Normalize timing-dependent output from stderr
	normalized := snapshot

	var filteredStderr []string
	for _, line := range snapshot.StderrLines {
		if line == "" {
			continue
		}
		// Filter out timing information which varies between runs
		if strings.Contains(line, "Total time:") ||
			strings.Contains(line, "duration") ||
			strings.Contains(line, "Files found:") ||
			strings.Contains(line, "Processed:") {
			continue
		}
		filteredStderr = append(filteredStderr, line)
	}
	normalized.StderrLines = filteredStderr

	// Normalize JSON output timing fields
	if normalized.JSONOutput != nil && normalized.JSONOutput.Metadata != nil {
		// We don't normalize metadata timing fields since they should be stable enough
	}

	return normalized
}

func (st *CLISnapshotTester) snapshotsEqual(a, b CLISnapshot) bool {
	// Basic field comparison
	if a.TestName != b.TestName ||
		a.Directory != b.Directory ||
		a.ExitCode != b.ExitCode ||
		len(a.Args) != len(b.Args) ||
		len(a.StdoutLines) != len(b.StdoutLines) ||
		len(a.StderrLines) != len(b.StderrLines) {
		return false
	}

	// Compare args
	for i, arg := range a.Args {
		if arg != b.Args[i] {
			return false
		}
	}

	// Compare stdout lines
	for i, line := range a.StdoutLines {
		if line != b.StdoutLines[i] {
			return false
		}
	}

	// Compare stderr lines
	for i, line := range a.StderrLines {
		if line != b.StderrLines[i] {
			return false
		}
	}

	// Compare JSON output if present
	if (a.JSONOutput == nil) != (b.JSONOutput == nil) {
		return false
	}

	if a.JSONOutput != nil {
		aJSON, _ := json.Marshal(a.JSONOutput)
		bJSON, _ := json.Marshal(b.JSONOutput)
		if string(aJSON) != string(bJSON) {
			return false
		}
	}

	return true
}

// Helper functions
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func filterEmptyLines(lines []string) []string {
	var result []string
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			result = append(result, line)
		}
	}
	return result
}

func (st *CLISnapshotTester) generateColoredDiff(expected, actual string) string {
	dmp := diffmatchpatch.New()

	// Generate diffs
	diffs := dmp.DiffMain(expected, actual, false)
	diffs = dmp.DiffCleanupSemantic(diffs)

	// ANSI color codes
	const (
		red   = "\033[31m"
		green = "\033[32m"
		reset = "\033[0m"
	)

	var result strings.Builder
	result.WriteString("\n--- Expected\n+++ Actual\n")

	// Track line numbers for both expected and actual
	expectedLineNum := 1
	actualLineNum := 1

	for _, diff := range diffs {
		switch diff.Type {
		case diffmatchpatch.DiffDelete:
			lines := strings.Split(diff.Text, "\n")
			for i, line := range lines {
				if line != "" || i < len(lines)-1 { // Include empty lines except final one
					result.WriteString(fmt.Sprintf("%s-%4d %s%s\n", red, expectedLineNum, line, reset))
					expectedLineNum++
				}
			}
		case diffmatchpatch.DiffInsert:
			lines := strings.Split(diff.Text, "\n")
			for i, line := range lines {
				if line != "" || i < len(lines)-1 { // Include empty lines except final one
					result.WriteString(fmt.Sprintf("%s+%4d %s%s\n", green, actualLineNum, line, reset))
					actualLineNum++
				}
			}
		case diffmatchpatch.DiffEqual:
			lines := strings.Split(diff.Text, "\n")
			// Only show a few lines of context around changes
			if len(lines) > 6 {
				// Show first 3 lines
				for i := 0; i < 3 && i < len(lines); i++ {
					if lines[i] != "" || i < len(lines)-1 {
						result.WriteString(fmt.Sprintf(" %4d %s\n", expectedLineNum, lines[i]))
						expectedLineNum++
						actualLineNum++
					}
				}

				// Skip middle lines and update counters
				skippedLines := len(lines) - 6
				if skippedLines > 0 {
					result.WriteString(fmt.Sprintf("      ... (%d lines)\n", skippedLines))
					expectedLineNum += skippedLines
					actualLineNum += skippedLines
				}

				// Show last 3 lines
				start := len(lines) - 3
				if start < 3 {
					start = 3
				}
				for i := start; i < len(lines); i++ {
					if lines[i] != "" || i < len(lines)-1 {
						result.WriteString(fmt.Sprintf(" %4d %s\n", expectedLineNum, lines[i]))
						expectedLineNum++
						actualLineNum++
					}
				}
			} else {
				for i, line := range lines {
					if line != "" || i < len(lines)-1 {
						result.WriteString(fmt.Sprintf(" %4d %s\n", expectedLineNum, line))
						expectedLineNum++
						actualLineNum++
					}
				}
			}
		}
	}

	return result.String()
}

func IsCLIBaselineMode() bool {
	return os.Getenv("UPDATE_SNAPSHOTS") == "1"
}
