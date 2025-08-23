package tests

import (
	"path/filepath"
	"testing"

	"github.com/ogdakke/symbolista/internal/snapshot"
)

func TestSnapshots(t *testing.T) {
	testDir := "./test_dir"
	snapshotDir := "./snapshots"

	baselineMode := snapshot.IsBaselineMode()
	tester := snapshot.NewSnapshotTester(snapshotDir, baselineMode)

	tests := []struct {
		name    string
		options snapshot.TestOptions
	}{
		{
			name: "basic_analysis",
			options: snapshot.TestOptions{
				IncludeDotfiles: false,
				ASCIIOnly:       true,
				OutputFormat:    "table",
				WorkerCount:     1,
			},
		},
		{
			name: "include_dotfiles",
			options: snapshot.TestOptions{
				IncludeDotfiles: true,
				ASCIIOnly:       true,
				OutputFormat:    "table",
				WorkerCount:     1,
			},
		},
		{
			name: "json_output",
			options: snapshot.TestOptions{
				IncludeDotfiles: false,
				ASCIIOnly:       true,
				OutputFormat:    "json",
				WorkerCount:     1,
			},
		},
		{
			name: "unicode_enabled",
			options: snapshot.TestOptions{
				IncludeDotfiles: false,
				ASCIIOnly:       false,
				OutputFormat:    "table",
				WorkerCount:     1,
			},
		},
		{
			name: "concurrent_processing",
			options: snapshot.TestOptions{
				IncludeDotfiles: false,
				ASCIIOnly:       true,
				OutputFormat:    "table",
				WorkerCount:     4,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tester.Test(t, tt.name, testDir, tt.options)
		})
	}
}

func TestSnapshotValidation(t *testing.T) {
	// Test that our test directory has the expected structure
	testDir := "./test_dir"

	// Check if test files exist
	expectedFiles := []string{
		"main.go",
		"config.json",
		"README.md",
		"data.csv",
		"src/utils.go",
		".gitignore",
		"ignored.log",
		"image.svg",
	}

	for _, file := range expectedFiles {
		fullPath := filepath.Join(testDir, file)
		if _, err := filepath.Glob(fullPath); err != nil {
			t.Errorf("Expected test file %s not found: %v", fullPath, err)
		}
	}
}
