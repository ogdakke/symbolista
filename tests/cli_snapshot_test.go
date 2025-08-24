package tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ogdakke/symbolista/internal/snapshot"
)

func TestCLISnapshots(t *testing.T) {
	testDir := "./test_dir"
	snapshotDir := "./cli_snapshots"

	binaryPath := "../tmp/symbolista"

	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Skip("Binary not found. Run 'make build' first.")
	}

	baselineMode := snapshot.IsCLIBaselineMode()
	tester := snapshot.NewCLISnapshotTester(snapshotDir, baselineMode, binaryPath)

	tests := []struct {
		name string
		args []string
	}{
		{
			name: "basic_analysis_table",
			args: []string{"--format=table"},
		},
		{
			name: "basic_analysis_json",
			args: []string{"--format=json", "--metadata=false"},
		},
		{
			name: "basic_analysis_csv",
			args: []string{"--format=csv"},
		},
		{
			name: "include_dotfiles_table",
			args: []string{"--format=table", "--include-dotfiles"},
		},
		{
			name: "include_dotfiles_json",
			args: []string{"--format=json", "--include-dotfiles", "--metadata=false"},
		},
		{
			name: "unicode_enabled_table",
			args: []string{"--format=table", "--ascii-only=false"},
		},
		{
			name: "unicode_enabled_json",
			args: []string{"--format=json", "--ascii-only=false", "--metadata=false"},
		},
		{
			name: "concurrent_processing_table",
			args: []string{"--format=table", "--workers=4"},
		},
		{
			name: "concurrent_processing_json",
			args: []string{"--format=json", "--workers=4", "--metadata=false"},
		},
		{
			name: "no_percentages_table",
			args: []string{"--format=table", "--percentages=false"},
		},
		{
			name: "no_percentages_json",
			args: []string{"--format=json", "--percentages=false", "--metadata=false"},
		},
		{
			name: "sequence_analysis_json",
			args: []string{"--format=json", "--metadata=false"},
		},
		{
			name: "sequence_analysis_table",
			args: []string{"--format=table"},
		},
		{
			name: "sequence_analysis_csv",
			args: []string{"--format=csv"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tester.Test(t, tt.name, testDir, tt.args)
		})
	}
}

func TestCLISnapshotValidation(t *testing.T) {
	testDir := "./test_dir"

	expectedFiles := []string{
		"main.go",
		"config.json",
		"README.md",
		"data.csv",
		"src/utils.go",
		"src/errors.ts",
		"src/test.js",
		"src/services/user.service.ts",
		"src/foo/some.tsx",
		".gitignore",
		"image.svg",
	}

	for _, file := range expectedFiles {
		fullPath := filepath.Join(testDir, file)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			t.Errorf("Expected test file %s not found", fullPath)
		}
	}
}
