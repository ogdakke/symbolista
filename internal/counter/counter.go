package counter

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/ogdakke/symbolista/internal/gitignore"
	"github.com/ogdakke/symbolista/internal/logger"
	"github.com/ogdakke/symbolista/internal/traversal"
)

type CharCount struct {
	Char       rune    `json:"char"`
	Count      int     `json:"count"`
	Percentage float64 `json:"percentage"`
}

type CharCounts []CharCount

func (c CharCounts) Len() int           { return len(c) }
func (c CharCounts) Less(i, j int) bool { return c[i].Count > c[j].Count }
func (c CharCounts) Swap(i, j int)      { c[i], c[j] = c[j], c[i] }

func CountSymbols(directory, format string, showPercentages bool) {
	CountSymbolsConcurrent(directory, format, showPercentages, 0)
}

func CountSymbolsConcurrent(directory, format string, showPercentages bool, workerCount int) {
	startTime := time.Now()

	logger.Info("Initializing gitignore matcher", "directory", directory)
	gitignoreStart := time.Now()
	matcher, err := gitignore.NewMatcher(directory)
	gitignoreDuration := time.Since(gitignoreStart)

	if err != nil {
		logger.Error("Could not load gitignore", "error", err, "duration", gitignoreDuration)
		fmt.Printf("Warning: Could not load gitignore: %v\n", err)
	} else {
		logger.Debug("Gitignore matcher created successfully", "duration", gitignoreDuration)
	}

	logger.Info("Starting concurrent file traversal and character counting")
	traversalStart := time.Now()

	result, err := traversal.WalkDirectoryConcurrent(directory, matcher, workerCount)
	traversalDuration := time.Since(traversalStart)

	if err != nil {
		logger.Error("Error during file processing", "error", err, "duration", traversalDuration)
		fmt.Printf("Error processing files: %v\n", err)
		return
	}

	charMap := result.CharMap
	totalChars := result.TotalChars
	processedFiles := result.FileCount

	logger.Info("File processing completed",
		"files_processed", processedFiles,
		"total_characters", totalChars,
		"unique_characters", len(charMap),
		"traversal_duration", traversalDuration)

	sortingStart := time.Now()
	var counts CharCounts
	for char, count := range charMap {
		percentage := float64(count) / float64(totalChars) * 100
		counts = append(counts, CharCount{
			Char:       char,
			Count:      count,
			Percentage: percentage,
		})
	}

	sort.Sort(counts)
	sortingDuration := time.Since(sortingStart)
	logger.Debug("Character counts sorted", "unique_chars", len(counts), "duration", sortingDuration)

	outputStart := time.Now()
	switch format {
	case "json":
		logger.Debug("Outputting results as JSON")
		outputJSON(counts, showPercentages)
	case "csv":
		logger.Debug("Outputting results as CSV")
		outputCSV(counts, showPercentages)
	default:
		logger.Debug("Outputting results as table")
		outputTable(counts, showPercentages)
	}
	outputDuration := time.Since(outputStart)

	totalDuration := time.Since(startTime)
	logger.Info("Analysis completed",
		"total_duration", totalDuration,
		"gitignore_duration", gitignoreDuration,
		"traversal_duration", traversalDuration,
		"sorting_duration", sortingDuration,
		"output_duration", outputDuration)
}

func outputTable(counts CharCounts, showPercentages bool) {
	fmt.Printf("%-10s %-10s", "Character", "Count")
	if showPercentages {
		fmt.Printf(" %-12s", "Percentage")
	}
	fmt.Println()
	fmt.Println(strings.Repeat("-", 35))

	for _, c := range counts {
		char := string(c.Char)
		switch c.Char {
		case ' ':
			char = "<space>"
		case '\t':
			char = "<tab>"
		case '\n':
			char = "<newline>"
		case '\r':
			char = "<return>"
		}

		fmt.Printf("%-10s %-10d", char, c.Count)
		if showPercentages {
			fmt.Printf(" %-12.2f%%", c.Percentage)
		}
		fmt.Println()
	}
}

func outputJSON(counts CharCounts, showPercentages bool) {
	if !showPercentages {
		for i := range counts {
			counts[i].Percentage = 0
		}
	}

	data, err := json.MarshalIndent(counts, "", "  ")
	if err != nil {
		fmt.Printf("Error marshaling JSON: %v\n", err)
		return
	}
	fmt.Println(string(data))
}

func outputCSV(counts CharCounts, showPercentages bool) {
	writer := csv.NewWriter(os.Stdout)
	defer writer.Flush()

	headers := []string{"Character", "Count"}
	if showPercentages {
		headers = append(headers, "Percentage")
	}
	writer.Write(headers)

	for _, c := range counts {
		char := string(c.Char)
		if c.Char == ' ' {
			char = "<space>"
		} else if c.Char == '\t' {
			char = "<tab>"
		} else if c.Char == '\n' {
			char = "<newline>"
		} else if c.Char == '\r' {
			char = "<return>"
		}

		row := []string{char, fmt.Sprintf("%d", c.Count)}
		if showPercentages {
			row = append(row, fmt.Sprintf("%.2f%%", c.Percentage))
		}
		writer.Write(row)
	}
}
