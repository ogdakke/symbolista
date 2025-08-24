package counter

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/ogdakke/symbolista/internal/concurrent"
	"github.com/ogdakke/symbolista/internal/domain"
	"github.com/ogdakke/symbolista/internal/ignorer"
	"github.com/ogdakke/symbolista/internal/logger"
	"github.com/ogdakke/symbolista/internal/output"
	"github.com/ogdakke/symbolista/internal/traversal"
)

func AnalyzeSymbols(
	directory string,
	workerCount int,
	includeDotfiles bool,
	asciiOnly bool,
	sequenceConfig concurrent.SequenceConfig,
	progressCallback func(filesFound, filesProcessed int),
	topNSeq int,
) (domain.AnalysisResult, error) {
	startTime := time.Now()

	logger.Info("Initializing gitignore matcher", "directory", directory, "includeDotfiles", includeDotfiles)
	matcher, err := ignorer.NewTimingMatcher(directory, includeDotfiles)

	if err != nil {
		logger.Error("Could not load gitignore", "error", err)
		return domain.AnalysisResult{}, fmt.Errorf("could not load gitignore: %w", err)
	} else {
		logger.Debug("Gitignore matcher created successfully", "initial_duration", matcher.GetLoadTime())
	}

	logger.Info("Starting concurrent file traversal and character counting")
	traversalStart := time.Now()

	result, err := traversal.WalkDirectoryConcurrent(directory, matcher.Matcher, workerCount, asciiOnly, sequenceConfig, progressCallback)
	traversalDuration := time.Since(traversalStart)

	if err != nil {
		logger.Error("Error during file processing", "error", err, "duration", traversalDuration)
		return domain.AnalysisResult{}, fmt.Errorf("error processing files: %w", err)
	}

	gitignoreDuration := matcher.GetTotalTime()

	charMap := result.CharMap
	sequenceMap2 := result.SequenceMap2
	sequenceMap3 := result.SequenceMap3

	// Convert uint16/uint32 keys back to strings and combine
	sequenceMap := make(map[string]int)
	for k2, count := range sequenceMap2 {
		seq := string([]byte{byte(k2 >> 8), byte(k2)})
		sequenceMap[seq] = int(count)
	}
	for k3, count := range sequenceMap3 {
		seq := string([]byte{byte(k3 >> 16), byte(k3 >> 8), byte(k3)})
		sequenceMap[seq] = int(count)
	}
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
	var counts domain.CharCounts
	for char, count := range charMap {
		percentage := float64(count) / float64(totalChars) * 100
		counts = append(counts, domain.CharCount{
			Char:       strings.ToLower(string(char)),
			Count:      count,
			Percentage: percentage,
		})
	}
	sort.Sort(counts)

	// Process sequence counts
	var sequenceCounts domain.SequenceCounts
	totalSequences := 0
	for _, count := range sequenceMap {
		totalSequences += count
	}

	for sequence, count := range sequenceMap {
		if count >= sequenceConfig.Threshold {
			percentage := float64(count) / float64(totalSequences) * 100
			sequenceCounts = append(sequenceCounts, domain.SequenceCount{
				Sequence:   sequence,
				Count:      count,
				Percentage: percentage,
			})
		}
	}
	sort.Sort(sequenceCounts)

	// Limit sequences to top N if specified
	if topNSeq > 0 && len(sequenceCounts) > topNSeq {
		sequenceCounts = sequenceCounts[:topNSeq]
	}

	sortingDuration := time.Since(sortingStart)
	logger.Debug("Counts sorted", "unique_chars", len(counts), "unique_sequences", len(sequenceCounts), "duration", sortingDuration)

	totalDuration := time.Since(startTime)

	timing := domain.TimingBreakdown{
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

	return domain.AnalysisResult{
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

func CountSymbolsConcurrent(
	outputter *output.Outputter,
	directory, format string,
	showPercentages bool,
	workerCount int,
	includeDotfiles bool,
	asciiOnly bool,
	includeMetadata bool,
	topNSeq int,
	countSequences bool,
) {

	var progressFunc func(int, int)

	progressFunc = func(filesFound, filesProcessed int) {
		fmt.Fprintf(os.Stderr, "\rFiles found: %d, Processed: %d", filesFound, filesProcessed)
	}

	sequenceConfig := concurrent.SequenceConfig{
		Enabled:   countSequences,
		MinLength: 2,
		MaxLength: 3,
		Threshold: 2,
	}

	result, err := AnalyzeSymbols(directory, workerCount, includeDotfiles, asciiOnly, sequenceConfig, progressFunc, topNSeq)

	fmt.Fprintf(os.Stderr, "\n")

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	outputStart := time.Now()
	outputter.Output(format, result, showPercentages, directory, includeMetadata)
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
