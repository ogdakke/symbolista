package traversal

import (
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/ogdakke/symbolista/internal/gitignore"
)

type FileProcessor func(path string, content []byte) error

func WalkDirectory(rootPath string, matcher *gitignore.Matcher, processor FileProcessor) error {
	return filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Skip symlinks and special files
		if info.Mode()&os.ModeType != 0 {
			return nil
		}

		if matcher != nil && matcher.ShouldIgnore(path) {
			return nil
		}

		// Skip binary files by checking file extension
		ext := strings.ToLower(filepath.Ext(path))
		binaryExtensions := []string{".exe", ".dll", ".so", ".dylib", ".bin", ".o", ".a", ".zip", ".tar", ".gz", ".jpg", ".jpeg", ".png", ".gif", ".pdf", ".mp4", ".mp3", ".avi"}
		if slices.Contains(binaryExtensions, ext) {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			// Skip files we can't read instead of failing
			return nil
		}
		defer file.Close()

		content, err := io.ReadAll(file)
		if err != nil {
			// Skip files we can't read instead of failing
			return nil
		}

		// Skip files that are not valid UTF-8 text
		if !utf8.Valid(content) {
			return nil
		}

		return processor(path, content)
	})
}
