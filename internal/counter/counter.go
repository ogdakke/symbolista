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
	Char       string  `json:"char"`
	Count      int     `json:"count"`
	Percentage float64 `json:"percentage"`
}

type CharCounts []CharCount

func (c CharCounts) Len() int           { return len(c) }
func (c CharCounts) Less(i, j int) bool { return c[i].Count > c[j].Count }
func (c CharCounts) Swap(i, j int)      { c[i], c[j] = c[j], c[i] }

type TimingBreakdown struct {
	TotalDuration     time.Duration `json:"total_duration"`
	GitignoreDuration time.Duration `json:"gitignore_duration"`
	TraversalDuration time.Duration `json:"traversal_duration"`
	SortingDuration   time.Duration `json:"sorting_duration"`
	OutputDuration    time.Duration `json:"output_duration"`
}

type AnalysisResult struct {
	CharCounts   CharCounts
	FilesFound   int
	FilesIgnored int
	TotalChars   int
	UniqueChars  int
	Timing       TimingBreakdown
}

type JSONMetadata struct {
	Directory       string          `json:"directory"`
	FilesFound      int             `json:"files_found"`
	FilesProcessed  int             `json:"files_processed"`
	FilesIgnored    int             `json:"files_ignored"`
	TotalCharacters int             `json:"total_characters"`
	UniqueChars     int             `json:"unique_characters"`
	Timing          TimingBreakdown `json:"timing"`
}

type JSONOutput struct {
	Result   CharCounts    `json:"result"`
	Metadata *JSONMetadata `json:"metadata,omitempty"`
}

func CountSymbols(directory, format string, showPercentages bool) {
	CountSymbolsConcurrent(directory, format, showPercentages, 0, false, true, false)
}

func AnalyzeSymbols(directory string, workerCount int, includeDotfiles bool, asciiOnly bool, progressCallback func(filesFound, filesProcessed int)) (AnalysisResult, error) {
	startTime := time.Now()

	logger.Info("Initializing gitignore matcher", "directory", directory, "includeDotfiles", includeDotfiles)
	matcher, err := gitignore.NewTimingMatcher(directory, includeDotfiles)

	if err != nil {
		logger.Error("Could not load gitignore", "error", err)
		return AnalysisResult{}, fmt.Errorf("could not load gitignore: %w", err)
	} else {
		logger.Debug("Gitignore matcher created successfully", "initial_duration", matcher.GetLoadTime())
	}

	logger.Info("Starting concurrent file traversal and character counting")
	traversalStart := time.Now()

	result, err := traversal.WalkDirectoryConcurrent(directory, matcher.Matcher, workerCount, asciiOnly, progressCallback)
	traversalDuration := time.Since(traversalStart)

	if err != nil {
		logger.Error("Error during file processing", "error", err, "duration", traversalDuration)
		return AnalysisResult{}, fmt.Errorf("error processing files: %w", err)
	}

	gitignoreDuration := matcher.GetTotalTime()

	charMap := result.CharMap
	totalChars := result.TotalChars
	processedFiles := result.FileCount
	filesFound := result.FilesFound
	filesIgnored := result.FilesIgnored

	logger.Info("File processing completed",
		"files_found", filesFound,
		"files_processed", processedFiles,
		"files_ignored", filesIgnored,
		"total_characters", totalChars,
		"unique_characters", len(charMap),
		"traversal_duration", traversalDuration)

	sortingStart := time.Now()
	var counts CharCounts
	for char, count := range charMap {
		percentage := float64(count) / float64(totalChars) * 100
		counts = append(counts, CharCount{
			Char:       strings.ToLower(string(char)),
			Count:      count,
			Percentage: percentage,
		})
	}

	sort.Sort(counts)
	sortingDuration := time.Since(sortingStart)
	logger.Debug("Character counts sorted", "unique_chars", len(counts), "duration", sortingDuration)

	totalDuration := time.Since(startTime)

	timing := TimingBreakdown{
		TotalDuration:     totalDuration,
		GitignoreDuration: gitignoreDuration,
		TraversalDuration: traversalDuration,
		SortingDuration:   sortingDuration,
		OutputDuration:    0, // Will be set by the caller
	}

	logger.Info("Analysis completed",
		"total_duration", totalDuration,
		"gitignore_duration", gitignoreDuration,
		"traversal_duration", traversalDuration,
		"sorting_duration", sortingDuration)

	return AnalysisResult{
		CharCounts:   counts,
		FilesFound:   filesFound,
		FilesIgnored: filesIgnored,
		TotalChars:   totalChars,
		UniqueChars:  len(charMap),
		Timing:       timing,
	}, nil
}

func CountSymbolsConcurrent(directory, format string, showPercentages bool, workerCount int, includeDotfiles bool, asciiOnly bool, includeMetadata bool) {

	var progressFunc func(int, int)
	if format == "table" {
		progressFunc = func(filesFound, filesProcessed int) {
			fmt.Printf("\rFiles found: %d, Processed: %d", filesFound, filesProcessed)
		}
	}

	result, err := AnalyzeSymbols(directory, workerCount, includeDotfiles, asciiOnly, progressFunc)
	if format == "table" && progressFunc != nil {
		fmt.Printf("\n")
	}
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	outputStart := time.Now()
	switch format {
	case "json":
		logger.Debug("Outputting results as JSON")
		outputJSON(result.CharCounts, showPercentages, directory, result, includeMetadata)
		return // Don't print summary for JSON format
	case "csv":
		logger.Debug("Outputting results as CSV")
		outputCSV(result.CharCounts, showPercentages)
	default:
		logger.Debug("Outputting results as table")
		outputTable(result.CharCounts, showPercentages)
	}
	outputDuration := time.Since(outputStart)

	// Update timing with output duration
	result.Timing.OutputDuration = outputDuration
	totalDuration := result.Timing.TotalDuration + outputDuration

	fmt.Printf("Files found: %d\n", result.FilesFound)
	fmt.Printf("Files processed: %d\n", result.FilesFound-result.FilesIgnored)
	fmt.Printf("Files/directories ignored: %d\n", result.FilesIgnored)
	fmt.Printf("Total characters: %d\n", result.TotalChars)
	fmt.Printf("Unique characters: %d\n", result.UniqueChars)

	if logger.GetVerbosity() > 0 {
		fmt.Println("\nTiming Breakdown:")
		fmt.Printf("  Gitignore initialization: %s\n", result.Timing.GitignoreDuration)
		fmt.Printf("  File traversal & counting: %s\n", result.Timing.TraversalDuration)
		fmt.Printf("  Sorting results: %s\n", result.Timing.SortingDuration)
		fmt.Printf("  Output formatting: %s\n", result.Timing.OutputDuration)
	}
	fmt.Printf("Total time: %s\n", totalDuration)
}

func outputTable(counts CharCounts, showPercentages bool) {
	fmt.Println(strings.Repeat("-", 35))
	fmt.Printf("%-10s %-10s", "Character", "Count")
	if showPercentages {
		fmt.Printf(" %-12s", "Percentage")
	}
	fmt.Println()
	fmt.Println(strings.Repeat("-", 35))

	formatChars(counts, func(char string, count int, percentage float64) {
		fmt.Printf("%-10s %-10d", char, count)
		if showPercentages {
			fmt.Printf(" %-12.2f%%", percentage)
		}
		fmt.Println()
	})
	fmt.Println(strings.Repeat("-", 35))
}

func outputJSON(counts CharCounts, showPercentages bool, directory string, result AnalysisResult, includeMetadata bool) {
	if !showPercentages {
		for i := range counts {
			counts[i].Percentage = 0
		}
	}

	output := JSONOutput{
		Result: counts,
	}

	if includeMetadata {
		output.Metadata = &JSONMetadata{
			Directory:       directory,
			FilesFound:      result.FilesFound,
			FilesProcessed:  result.FilesFound - result.FilesIgnored,
			FilesIgnored:    result.FilesIgnored,
			TotalCharacters: result.TotalChars,
			UniqueChars:     result.UniqueChars,
			Timing:          result.Timing,
		}
	}

	data, err := json.MarshalIndent(output, "", "  ")
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

	formatChars(counts, func(char string, count int, percentage float64) {
		row := []string{char, fmt.Sprintf("%d", count)}
		if showPercentages {
			row = append(row, fmt.Sprintf("%.2f%%", percentage))
		}
		writer.Write(row)
	})
}

type OnCharFunc func(char string, count int, percentage float64)

func formatChars(counts CharCounts, onChar OnCharFunc) CharCounts {
	for _, c := range counts {
		char := c.Char
		switch c.Char {
		case " ":
			char = "<space>"
		case "\t":
			char = "<tab>"
		case "\n":
			char = "<newline>"
		case "\r":
			char = "<return>"
		case "\f":
			char = "<formfeed>"
		case "\v":
			char = "<vert_tab>"
		}

		onChar(char, c.Count, c.Percentage)
	}
	return counts
}
