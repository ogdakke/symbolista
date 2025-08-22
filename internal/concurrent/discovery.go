package concurrent

import (
	"io"
	"os"
	"path/filepath"
	"unicode/utf8"

	"github.com/ogdakke/symbolista/internal/gitignore"
	"github.com/ogdakke/symbolista/internal/logger"
)

func DiscoverFiles(rootPath string, matcher *gitignore.Matcher, jobChan chan<- FileJob, errorCallback func(error)) {
	defer close(jobChan)

	logger.Debug("Starting file discovery", "root_path", rootPath)

	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if errorCallback != nil {
				errorCallback(err)
			}
			return nil // Continue processing other files
		}

		// Handle directories
		if info.IsDir() {
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

		// Skip symlinks and special files
		if info.Mode()&os.ModeType != 0 {
			logger.Debug("Skipping special file", "path", path, "mode", info.Mode().String())
			return nil
		}

		// Skip ignored files
		if matcher != nil && matcher.ShouldIgnore(path) {
			logger.Debug("Skipping file (gitignore)", "path", path)
			return nil
		}

		// Read file content
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

		logger.Trace("Discovered file", "path", path, "size", len(content))

		// Send job to worker pool
		job := FileJob{
			Path:    path,
			Content: content,
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
