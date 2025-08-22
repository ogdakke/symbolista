package traversal

import (
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/ogdakke/symbolista/internal/gitignore"
	"github.com/ogdakke/symbolista/internal/logger"
)

type FileProcessor func(path string, content []byte) error

func WalkDirectory(rootPath string, matcher *gitignore.Matcher, processor FileProcessor) error {
	return filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			logger.Trace("Skipping directory", "path", path)
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

		// Skip binary files by checking file extension
		ext := strings.ToLower(filepath.Ext(path))
		binaryExtensions := []string{".exe", ".dll", ".so", ".dylib", ".bin", ".o", ".a", ".zip", ".tar", ".gz", ".jpg", ".jpeg", ".png", ".gif", ".pdf", ".mp4", ".mp3", ".avi"}
		if slices.Contains(binaryExtensions, ext) {
			logger.Debug("Skipping binary file", "path", path, "extension", ext)
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
