package concurrent

import (
	"io"
	"os"
	"path/filepath"
	"unicode/utf8"

	"github.com/ogdakke/symbolista/internal/gitignore"
	"github.com/ogdakke/symbolista/internal/logger"
)

func DiscoverFiles(rootPath string, matcher *gitignore.Matcher, jobChan chan<- FileJob, asciiOnly bool, collector *ResultCollector, progressCallback ProgressCallback, errorCallback func(error)) {
	defer close(jobChan)

	logger.Debug("Starting file discovery", "root_path", rootPath)

	err := filepath.WalkDir(rootPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			if errorCallback != nil {
				errorCallback(err)
			}
			return nil // Continue processing other files
		}

		// Handle directories
		if d.IsDir() {
			// Load gitignore file if it exists in this directory
			if matcher != nil {
				if err := matcher.LoadGitignoreForDirectory(path); err != nil {
					logger.Debug("Error loading gitignore", "path", path, "error", err)
				}
			}

			// Don't traverse into ignored directories
			if path != rootPath && matcher != nil && matcher.ShouldIgnore(path) {
				logger.Debug("Skipping directory (gitignore)", "path", path)
				return filepath.SkipDir
			}
			logger.Trace("Entering directory", "path", path)
			return nil
		}

		// Count all regular files found
		collector.IncrementFound()

		// Report progress if callback provided
		if progressCallback != nil {
			_, _, _, filesFound, filesIgnored := collector.GetResults()
			progressCallback(filesFound, filesFound-filesIgnored)
		}

		// Skip symlinks and special files
		if d.Type()&os.ModeType != 0 {
			logger.Debug("Skipping special file", "path", path, "mode", d.Type().String())
			collector.IncrementIgnored()
			return nil
		}

		// Skip ignored files
		if matcher != nil && matcher.ShouldIgnore(path) {
			logger.Debug("Skipping file (gitignore)", "path", path)
			collector.IncrementIgnored()
			return nil
		}

		// Read file content
		file, err := os.Open(path)
		if err != nil {
			logger.Debug("Cannot read file", "path", path, "error", err)
			collector.IncrementIgnored()
			return nil
		}
		defer file.Close()

		content, err := io.ReadAll(file)
		if err != nil {
			logger.Debug("Cannot read file content", "path", path, "error", err)
			collector.IncrementIgnored()
			return nil
		}

		// Skip files that are not valid UTF-8 text
		if !utf8.Valid(content) {
			logger.Debug("Skipping non-UTF8 file", "path", path)
			collector.IncrementIgnored()
			return nil
		}

		logger.Trace("Discovered file", "path", path, "size", len(content))

		// Send job to worker pool
		job := FileJob{
			Path:      path,
			Content:   content,
			AsciiOnly: asciiOnly,
		}

		select {
		case jobChan <- job:
			// Job sent successfully
		default:
			// Channel is full, this shouldn't happen with proper buffer sizing
			logger.Debug("Job channel full, this may indicate a bottleneck", "path", path)
			jobChan <- job // Block until space is available
		}

		return nil
	})

	if err != nil && errorCallback != nil {
		errorCallback(err)
	}

	logger.Debug("File discovery completed")
}
