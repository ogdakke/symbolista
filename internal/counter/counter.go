package counter

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"unicode"

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
	logger.Info("Initializing gitignore matcher", "directory", directory)
	matcher, err := gitignore.NewMatcher(directory)
	if err != nil {
		logger.Error("Could not load gitignore", "error", err)
		fmt.Printf("Warning: Could not load gitignore: %v\n", err)
	} else {
		logger.Debug("Gitignore matcher created successfully")
	}

	charMap := make(map[rune]int)
	totalChars := 0
	processedFiles := 0

	logger.Info("Starting file traversal and character counting")
	err = traversal.WalkDirectory(directory, matcher, func(path string, content []byte) error {
		processedFiles++
		fileChars := 0
		for _, r := range string(content) {
			if unicode.IsGraphic(r) || unicode.IsSpace(r) {
				charMap[r]++
				totalChars++
				fileChars++
			}
		}
		logger.Trace("Processed file", "path", path, "chars", fileChars)
		return nil
	})

	if err != nil {
		logger.Error("Error during file processing", "error", err)
		fmt.Printf("Error processing files: %v\n", err)
		return
	}

	logger.Info("File processing completed", "files_processed", processedFiles, "total_characters", totalChars, "unique_characters", len(charMap))

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
	logger.Debug("Character counts sorted", "unique_chars", len(counts))

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
		if c.Char == ' ' {
			char = "<space>"
		} else if c.Char == '\t' {
			char = "<tab>"
		} else if c.Char == '\n' {
			char = "<newline>"
		} else if c.Char == '\r' {
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
