package traversal

import (
	"io"
	"os"
	"path/filepath"
	"runtime"
	"unicode/utf8"

	"github.com/ogdakke/symbolista/internal/concurrent"
	"github.com/ogdakke/symbolista/internal/ignorer"
	"github.com/ogdakke/symbolista/internal/logger"
)

type FileProcessor func(path string, content []byte) error

func WalkDirectory(rootPath string, matcher *ignorer.Matcher, processor FileProcessor) error {
	return filepath.WalkDir(rootPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			if matcher != nil {
				if err := matcher.LoadGitignoreForDirectory(path); err != nil {
					logger.Debug("Error loading gitignore", "path", path, "error", err)
				}
			}

			if path != rootPath && matcher != nil && matcher.ShouldIgnore(path) {
				logger.Debug("Skipping directory (gitignore)", "path", path)
				return filepath.SkipDir
			}
			logger.Trace("Entering directory", "path", path)
			return nil
		}

		if d.Type()&os.ModeType != 0 {
			logger.Debug("Skipping special file", "path", path, "mode", d.Type().String())
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

		if !utf8.Valid(content) {
			logger.Debug("Skipping non-UTF8 file", "path", path)
			return nil
		}

		logger.Trace("Processing file", "path", path, "size", len(content))

		return processor(path, content)
	})
}

type ConcurrentResult struct {
	CharMap          map[rune]int
	SequenceMap2     map[uint16]uint32
	SequenceMap3     map[uint32]uint32
	FileCount        int
	FilesFound       int
	FilesIgnored     int
	TotalChars       int
	UniqueChars      int
	UniqueSequences2 int
	UniqueSequences3 int
}

// WalkDirectoryConcurrent processes files using a worker pool and returns aggregated results
func WalkDirectoryConcurrent(
	rootPath string,
	matcher *ignorer.Matcher,
	workerCount int,
	asciiOnly bool,
	sequenceConfig concurrent.SequenceConfig,
	progressCallback concurrent.ProgressCallback,
) (ConcurrentResult, error) {
	if workerCount <= 0 {
		workerCount = runtime.NumCPU()
	}

	bufferSize := workerCount * 2

	pool := concurrent.NewWorkerPool(workerCount, bufferSize)
	collector := concurrent.NewResultCollector()

	pool.Start()

	var discoveryError error
	go concurrent.DiscoverFiles(rootPath, matcher, pool.Jobs(), asciiOnly, sequenceConfig, collector, progressCallback, func(err error) {
		if discoveryError == nil {
			discoveryError = err
		}
	})

	for result := range pool.Results() {
		collector.AddResult(result)
	}

	<-pool.Done()

	if discoveryError != nil {
		return ConcurrentResult{}, discoveryError
	}

	charMap, sequenceMap2, sequenceMap3, fileCount, totalChars, filesFound, filesIgnored, timing := collector.GetResults()

	logger.Info("Concurrent processing completed",
		"files_processed", fileCount,
		"files_found", filesFound,
		"files_ignored", filesIgnored,
		"total_characters", totalChars,
		"unique_characters", len(charMap),
		"workers", workerCount,
		"timing", timing,
	)

	return ConcurrentResult{
		CharMap:          charMap,
		SequenceMap2:     sequenceMap2,
		SequenceMap3:     sequenceMap3,
		FileCount:        fileCount,
		FilesFound:       filesFound,
		FilesIgnored:     filesIgnored,
		TotalChars:       totalChars,
		UniqueChars:      len(charMap),
		UniqueSequences2: len(sequenceMap2),
		UniqueSequences3: len(sequenceMap3),
	}, nil
}
