package snapshot

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ogdakke/symbolista/internal/concurrent"
	"github.com/ogdakke/symbolista/internal/counter"
)

type TestSnapshot struct {
	TestName   string                 `json:"test_name"`
	Directory  string                 `json:"directory"`
	Options    TestOptions            `json:"options"`
	Result     counter.AnalysisResult `json:"result"`
	JSONOutput *counter.JSONOutput    `json:"json_output,omitempty"`
}

type TestOptions struct {
	IncludeDotfiles bool   `json:"include_dotfiles"`
	ASCIIOnly       bool   `json:"ascii_only"`
	OutputFormat    string `json:"output_format"`
	WorkerCount     int    `json:"worker_count"`
}

type SnapshotTester struct {
	snapshotDir  string
	baselineMode bool
}

func NewSnapshotTester(snapshotDir string, baselineMode bool) *SnapshotTester {
	return &SnapshotTester{
		snapshotDir:  snapshotDir,
		baselineMode: baselineMode,
	}
}

func (st *SnapshotTester) Test(t *testing.T, testName string, testDir string, options TestOptions) {
	t.Helper()

	// Run the analysis
	result := st.runAnalysis(testDir, options)

	snapshot := TestSnapshot{
		TestName:  testName,
		Directory: testDir,
		Options:   options,
		Result:    result,
	}

	// Add JSON output if format is JSON
	if options.OutputFormat == "json" {
		jsonOutput := counter.JSONOutput{
			Result: counter.JSONResult{
				Characters: result.CharCounts,
				Sequences:  result.SequenceCounts,
			},
			Metadata: &counter.JSONMetadata{
				Directory:       testDir,
				FilesFound:      result.FilesFound,
				FilesProcessed:  result.FilesFound - result.FilesIgnored,
				FilesIgnored:    result.FilesIgnored,
				TotalCharacters: result.TotalChars,
				UniqueChars:     result.UniqueChars,
				Timing:          result.Timing,
			},
		}
		snapshot.JSONOutput = &jsonOutput
	}

	snapshotPath := filepath.Join(st.snapshotDir, testName+".json")

	if st.baselineMode {
		st.createSnapshot(t, snapshot, snapshotPath)
	} else {
		st.compareSnapshot(t, snapshot, snapshotPath)
	}
}

func (st *SnapshotTester) runAnalysis(testDir string, options TestOptions) counter.AnalysisResult {
	sequenceConfig := concurrent.SequenceConfig{
		Enabled:   false,
		MinLength: 2,
		MaxLength: 3,
		Threshold: 1,
	}
	result, err := counter.AnalyzeSymbols(testDir, options.WorkerCount, options.IncludeDotfiles, options.ASCIIOnly, sequenceConfig, nil)
	if err != nil {
		panic(fmt.Sprintf("Analysis failed: %v", err))
	}
	return result
}

func (st *SnapshotTester) createSnapshot(t *testing.T, snapshot TestSnapshot, snapshotPath string) {
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
		t.Fatalf("Failed to write snapshot: %v", err)
	}

	t.Logf("Created snapshot: %s", snapshotPath)
}

func (st *SnapshotTester) compareSnapshot(t *testing.T, snapshot TestSnapshot, snapshotPath string) {
	t.Helper()

	// Read existing snapshot
	data, err := os.ReadFile(snapshotPath)
	if err != nil {
		t.Fatalf("Failed to read snapshot %s: %v. Run with UPDATE_SNAPSHOTS=1 to create baseline.", snapshotPath, err)
	}

	var expectedSnapshot TestSnapshot
	if err := json.Unmarshal(data, &expectedSnapshot); err != nil {
		t.Fatalf("Failed to unmarshal snapshot: %v", err)
	}

	// Normalize both snapshots for comparison
	normalizedActual := st.normalizeSnapshot(snapshot)
	normalizedExpected := st.normalizeSnapshot(expectedSnapshot)

	// Compare the snapshots
	if !st.compareSnapshots(normalizedActual, normalizedExpected) {
		actualJSON, _ := json.MarshalIndent(normalizedActual, "", "  ")
		expectedJSON, _ := json.MarshalIndent(normalizedExpected, "", "  ")

		t.Errorf("Snapshot mismatch for %s\n\nActual:\n%s\n\nExpected:\n%s",
			snapshot.TestName, string(actualJSON), string(expectedJSON))
	}
}

func (st *SnapshotTester) normalizeSnapshot(snapshot TestSnapshot) TestSnapshot {
	// Create a copy and normalize timing-sensitive data
	normalized := snapshot

	// Zero out timing data as it's not deterministic
	normalized.Result.Timing.TotalDuration = 0
	normalized.Result.Timing.GitignoreDuration = 0
	normalized.Result.Timing.TraversalDuration = 0
	normalized.Result.Timing.SortingDuration = 0
	normalized.Result.Timing.OutputDuration = 0

	// Normalize JSON output timing if present
	if normalized.JSONOutput != nil {
		if normalized.JSONOutput.Metadata != nil {
			normalized.JSONOutput.Metadata.Timing.TotalDuration = 0
			normalized.JSONOutput.Metadata.Timing.GitignoreDuration = 0
			normalized.JSONOutput.Metadata.Timing.TraversalDuration = 0
			normalized.JSONOutput.Metadata.Timing.SortingDuration = 0
			normalized.JSONOutput.Metadata.Timing.OutputDuration = 0
		}
	}

	return normalized
}

func (st *SnapshotTester) compareSnapshots(actual, expected TestSnapshot) bool {
	// Compare test metadata
	if actual.TestName != expected.TestName ||
		actual.Options != expected.Options {
		return false
	}

	// Compare analysis results (excluding timing)
	if actual.Result.FilesFound != expected.Result.FilesFound ||
		actual.Result.FilesIgnored != expected.Result.FilesIgnored ||
		actual.Result.TotalChars != expected.Result.TotalChars ||
		actual.Result.UniqueChars != expected.Result.UniqueChars {
		return false
	}

	// Compare character counts
	if len(actual.Result.CharCounts) != len(expected.Result.CharCounts) {
		return false
	}

	for i, actualChar := range actual.Result.CharCounts {
		expectedChar := expected.Result.CharCounts[i]
		if actualChar.Char != expectedChar.Char ||
			actualChar.Count != expectedChar.Count ||
			fmt.Sprintf("%.2f", actualChar.Percentage) != fmt.Sprintf("%.2f", expectedChar.Percentage) {
			return false
		}
	}

	return true
}

func IsBaselineMode() bool {
	return os.Getenv("UPDATE_SNAPSHOTS") == "1" ||
		os.Getenv("BASELINE_MODE") == "1" ||
		strings.Contains(strings.Join(os.Args, " "), "-update-snapshots")
}
