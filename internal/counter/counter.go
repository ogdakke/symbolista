package counter

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/ogdakke/symbolista/internal/concurrent"
	"github.com/ogdakke/symbolista/internal/ignorer"
	"github.com/ogdakke/symbolista/internal/logger"
	"github.com/ogdakke/symbolista/internal/traversal"
)

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

type SequenceCounts []SequenceCount

func (s SequenceCounts) Len() int { return len(s) }
func (s SequenceCounts) Less(i, j int) bool {
	if s[i].Count != s[j].Count {
		return s[i].Count > s[j].Count
	}
	return s[i].Sequence < s[j].Sequence
}
func (s SequenceCounts) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

type CharCounts []CharCount

func (c CharCounts) Len() int { return len(c) }
func (c CharCounts) Less(i, j int) bool {
	if c[i].Count != c[j].Count {
		return c[i].Count > c[j].Count
	}
	return c[i].Char < c[j].Char
}
func (c CharCounts) Swap(i, j int) { c[i], c[j] = c[j], c[i] }

type TimingBreakdown struct {
	TotalDuration     time.Duration `json:"total_duration"`
	GitignoreDuration time.Duration `json:"gitignore_duration"`
	TraversalDuration time.Duration `json:"traversal_duration"`
	SortingDuration   time.Duration `json:"sorting_duration"`
	OutputDuration    time.Duration `json:"output_duration"`
}

type AnalysisResult struct {
	CharCounts      CharCounts
	SequenceCounts  SequenceCounts
	FilesFound      int
	FilesIgnored    int
	TotalChars      int
	UniqueChars     int
	UniqueSequences int
	Timing          TimingBreakdown
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

type JSONResult struct {
	Characters CharCounts     `json:"characters"`
	Sequences  SequenceCounts `json:"sequences"`
}

type JSONOutput struct {
	Result   JSONResult    `json:"result"`
	Metadata *JSONMetadata `json:"metadata,omitempty"`
}

func CountSymbols(directory, format string, showPercentages bool) {
	CountSymbolsConcurrent(directory, format, showPercentages, 0, false, true, false)
}

func AnalyzeSymbols(directory string, workerCount int, includeDotfiles bool, asciiOnly bool, sequenceConfig concurrent.SequenceConfig, progressCallback func(filesFound, filesProcessed int)) (AnalysisResult, error) {
	startTime := time.Now()

	logger.Info("Initializing gitignore matcher", "directory", directory, "includeDotfiles", includeDotfiles)
	matcher, err := ignorer.NewTimingMatcher(directory, includeDotfiles)

	if err != nil {
		logger.Error("Could not load gitignore", "error", err)
		return AnalysisResult{}, fmt.Errorf("could not load gitignore: %w", err)
	} else {
		logger.Debug("Gitignore matcher created successfully", "initial_duration", matcher.GetLoadTime())
	}

	logger.Info("Starting concurrent file traversal and character counting")
	traversalStart := time.Now()

	result, err := traversal.WalkDirectoryConcurrent(directory, matcher.Matcher, workerCount, asciiOnly, sequenceConfig, progressCallback)
	traversalDuration := time.Since(traversalStart)

	if err != nil {
		logger.Error("Error during file processing", "error", err, "duration", traversalDuration)
		return AnalysisResult{}, fmt.Errorf("error processing files: %w", err)
	}

	gitignoreDuration := matcher.GetTotalTime()

	charMap := result.CharMap
	sequenceMap := result.SequenceMap
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

	// Process character counts
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

	// Process sequence counts
	var sequenceCounts SequenceCounts
	totalSequences := 0
	for _, count := range sequenceMap {
		totalSequences += count
	}

	for sequence, count := range sequenceMap {
		percentage := float64(count) / float64(totalSequences) * 100
		sequenceCounts = append(sequenceCounts, SequenceCount{
			Sequence:   sequence,
			Count:      count,
			Percentage: percentage,
		})
	}
	sort.Sort(sequenceCounts)

	sortingDuration := time.Since(sortingStart)
	logger.Debug("Counts sorted", "unique_chars", len(counts), "unique_sequences", len(sequenceCounts), "duration", sortingDuration)

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
		CharCounts:      counts,
		SequenceCounts:  sequenceCounts,
		FilesFound:      filesFound,
		FilesIgnored:    filesIgnored,
		TotalChars:      totalChars,
		UniqueChars:     len(charMap),
		UniqueSequences: len(sequenceMap),
		Timing:          timing,
	}, nil
}

func CountSymbolsConcurrent(directory, format string, showPercentages bool, workerCount int, includeDotfiles bool, asciiOnly bool, includeMetadata bool) {

	var progressFunc func(int, int)

	progressFunc = func(filesFound, filesProcessed int) {
		fmt.Fprintf(os.Stderr, "\rFiles found: %d, Processed: %d", filesFound, filesProcessed)
	}

	// Default sequence config - enabled with reasonable threshold
	sequenceConfig := concurrent.SequenceConfig{
		Enabled:   true,
		MinLength: 2,
		MaxLength: 3,
		Threshold: 2,
	}

	result, err := AnalyzeSymbols(directory, workerCount, includeDotfiles, asciiOnly, sequenceConfig, progressFunc)

	fmt.Fprintf(os.Stderr, "\n")

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	outputStart := time.Now()
	switch format {
	case "json":
		logger.Debug("Outputting results as JSON")
		outputJSON(result.CharCounts, showPercentages, directory, result, includeMetadata)
	case "csv":
		logger.Debug("Outputting results as CSV")
		outputCSV(result.CharCounts, result.SequenceCounts, showPercentages)
	default:
		logger.Debug("Outputting results as table")
		outputTable(result.CharCounts, result.SequenceCounts, showPercentages)
	}
	outputDuration := time.Since(outputStart)

	result.Timing.OutputDuration = outputDuration
	totalDuration := result.Timing.TotalDuration + outputDuration

	fmt.Fprintf(os.Stderr, "Files/directories ignored: %d\n", result.FilesIgnored)
	fmt.Fprintf(os.Stderr, "Total characters: %d\n", result.TotalChars)
	fmt.Fprintf(os.Stderr, "Unique characters: %d\n", result.UniqueChars)

	if logger.GetVerbosity() > 0 {
		fmt.Fprintf(os.Stderr, "\nTiming Breakdown:\n")
		fmt.Fprintf(os.Stderr, "  Gitignore initialization: %s\n", result.Timing.GitignoreDuration)
		fmt.Fprintf(os.Stderr, "  File traversal & counting: %s\n", result.Timing.TraversalDuration)
		fmt.Fprintf(os.Stderr, "  Sorting results: %s\n", result.Timing.SortingDuration)
		fmt.Fprintf(os.Stderr, "  Output formatting: %s\n", result.Timing.OutputDuration)
	}
	fmt.Fprintf(os.Stderr, "Total time: %s\n", totalDuration)
}

func outputTable(counts CharCounts, sequences SequenceCounts, showPercentages bool) {
	// Characters table
	fmt.Println("Characters:")
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

	// Sequences table (if sequences exist)
	if len(sequences) > 0 {
		fmt.Printf("\nSequences (2-3 chars):\n")
		fmt.Println(strings.Repeat("-", 35))
		fmt.Printf("%-10s %-10s", "Sequence", "Count")
		if showPercentages {
			fmt.Printf(" %-12s", "Percentage")
		}
		fmt.Println()
		fmt.Println(strings.Repeat("-", 35))

		for _, seq := range sequences {
			fmt.Printf("%-10s %-10d", seq.Sequence, seq.Count)
			if showPercentages {
				fmt.Printf(" %-12.2f%%", seq.Percentage)
			}
			fmt.Println()
		}
		fmt.Println(strings.Repeat("-", 35))
	}
}

func outputJSON(counts CharCounts, showPercentages bool, directory string, result AnalysisResult, includeMetadata bool) {
	if !showPercentages {
		for i := range counts {
			counts[i].Percentage = 0
		}
	}

	output := JSONOutput{
		Result: JSONResult{
			Characters: counts,
			Sequences:  result.SequenceCounts,
		},
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

func outputCSV(counts CharCounts, sequences SequenceCounts, showPercentages bool) {
	writer := csv.NewWriter(os.Stdout)
	defer writer.Flush()

	headers := []string{"type", "sequence", "count"}
	if showPercentages {
		headers = append(headers, "percentage")
	}
	writer.Write(headers)

	// Write character data
	formatChars(counts, func(char string, count int, percentage float64) {
		row := []string{"character", char, fmt.Sprintf("%d", count)}
		if showPercentages {
			row = append(row, fmt.Sprintf("%.2f%%", percentage))
		}
		writer.Write(row)
	})

	// Write sequence data
	for _, seq := range sequences {
		row := []string{"sequence", seq.Sequence, fmt.Sprintf("%d", seq.Count)}
		if showPercentages {
			row = append(row, fmt.Sprintf("%.2f%%", seq.Percentage))
		}
		writer.Write(row)
	}
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
