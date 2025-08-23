package concurrent

import (
	"io"
	"os"
	"path/filepath"
	"unicode/utf8"

	"github.com/ogdakke/symbolista/internal/ignorer"
	"github.com/ogdakke/symbolista/internal/logger"
)

func DiscoverFiles(rootPath string, matcher *ignorer.Matcher, jobChan chan<- FileJob, asciiOnly bool, collector *ResultCollector, progressCallback ProgressCallback, errorCallback func(error)) {
	defer close(jobChan)

	logger.Debug("Starting file discovery", "root_path", rootPath)

	err := filepath.WalkDir(rootPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			if errorCallback != nil {
				errorCallback(err)
			}
			return nil
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

		collector.IncrementFound()

		if progressCallback != nil {
			_, _, _, filesFound, filesIgnored := collector.GetResults()
			progressCallback(filesFound, filesFound-filesIgnored)
		}

		if d.Type()&os.ModeType != 0 {
			logger.Debug("Skipping special file", "path", path, "mode", d.Type().String())
			collector.IncrementIgnored()
			return nil
		}

		if matcher != nil && matcher.ShouldIgnore(path) {
			logger.Debug("Skipping file (gitignore)", "path", path)
			collector.IncrementIgnored()
			return nil
		}

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

		if !utf8.Valid(content) {
			logger.Debug("Skipping non-UTF8 file", "path", path)
			collector.IncrementIgnored()
			return nil
		}

		logger.Trace("Discovered file", "path", path, "size", len(content))

		job := FileJob{
			Path:      path,
			Content:   content,
			AsciiOnly: asciiOnly,
		}

		select {
		case jobChan <- job:

		default:

			logger.Debug("Job channel full, this may indicate a bottleneck", "path", path)
			jobChan <- job
		}

		return nil
	})

	if err != nil && errorCallback != nil {
		errorCallback(err)
	}

	logger.Debug("File discovery completed")
}
