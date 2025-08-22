package traversal

import (
	"io"
	"os"
	"path/filepath"
	"runtime"
	"unicode/utf8"

	"github.com/ogdakke/symbolista/internal/concurrent"
	"github.com/ogdakke/symbolista/internal/gitignore"
	"github.com/ogdakke/symbolista/internal/logger"
)

type FileProcessor func(path string, content []byte) error

func WalkDirectory(rootPath string, matcher *gitignore.Matcher, processor FileProcessor) error {
	return filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Check if this directory should be ignored
		if info.IsDir() {
			// Load gitignore file if it exists in this directory
			if matcher != nil {
				if err := matcher.LoadGitignoreForDirectory(path); err != nil {
					logger.Debug("Error loading gitignore", "path", path, "error", err)
				}
			}

			// Don't traverse into the root directory
			if path != rootPath && matcher != nil && matcher.ShouldIgnore(path) {
				logger.Debug("Skipping directory (gitignore)", "path", path)
				return filepath.SkipDir
			}
			logger.Trace("Entering directory", "path", path)
			return nil
		}

		// Skip symlinks and special files
		if info.Mode()&os.ModeType != 0 {
			logger.Debug("Skipping special file", "path", path, "mode", info.Mode().String())
			return nil
		}

		if matcher != nil && matcher.ShouldIgnore(path) {
			logger.Debug("Skipping file (gitignore)", "path", path)
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			logger.Debug("Cannot read file", "path", path, "error", err)
			return nil
		}
		defer file.Close()

		content, err := io.ReadAll(file)
		if err != nil {
			logger.Debug("Cannot read file content", "path", path, "error", err)
			return nil
		}

		// Skip files that are not valid UTF-8 text
		if !utf8.Valid(content) {
			logger.Debug("Skipping non-UTF8 file", "path", path)
			return nil
		}

		logger.Trace("Processing file", "path", path, "size", len(content))

		return processor(path, content)
	})
}

// ConcurrentResult contains the results of concurrent file processing
type ConcurrentResult struct {
	CharMap     map[rune]int
	FileCount   int
	TotalChars  int
	UniqueChars int
}

// WalkDirectoryConcurrent processes files using a worker pool and returns aggregated results
func WalkDirectoryConcurrent(rootPath string, matcher *gitignore.Matcher, workerCount int) (ConcurrentResult, error) {
	if workerCount <= 0 {
		workerCount = runtime.NumCPU()
	}

	// Calculate buffer size based on worker count
	bufferSize := workerCount * 2

	// Create worker pool
	pool := concurrent.NewWorkerPool(workerCount, bufferSize)
	collector := concurrent.NewResultCollector()

	// Start worker pool
	pool.Start()

	// Start file discovery in a separate goroutine
	var discoveryError error
	go concurrent.DiscoverFiles(rootPath, matcher, pool.Jobs(), func(err error) {
		if discoveryError == nil {
			discoveryError = err
		}
	})

	// Collect results
	for result := range pool.Results() {
		collector.AddResult(result)
	}

	// Wait for completion
	<-pool.Done()

	if discoveryError != nil {
		return ConcurrentResult{}, discoveryError
	}

	// Get aggregated results
	charMap, fileCount, totalChars := collector.GetResults()

	logger.Debug("Concurrent processing completed",
		"files_processed", fileCount,
		"total_characters", totalChars,
		"unique_characters", len(charMap),
		"workers", workerCount)

	return ConcurrentResult{
		CharMap:     charMap,
		FileCount:   fileCount,
		TotalChars:  totalChars,
		UniqueChars: len(charMap),
	}, nil
}
