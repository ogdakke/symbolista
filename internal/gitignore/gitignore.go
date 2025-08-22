package gitignore

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"github.com/ogdakke/symbolista/internal/logger"
)

type Matcher struct {
	patterns []string
	basePath string
}

func NewMatcher(basePath string) (*Matcher, error) {
	matcher := &Matcher{
		basePath: basePath,
	}

	gitignorePath := filepath.Join(basePath, ".gitignore")
	if _, err := os.Stat(gitignorePath); os.IsNotExist(err) {
		logger.Debug("No .gitignore found", "path", gitignorePath)
		return matcher, nil
	}

	logger.Debug("Loading .gitignore", "path", gitignorePath)
	file, err := os.Open(gitignorePath)
	if err != nil {
		logger.Error("Cannot open .gitignore", "path", gitignorePath, "error", err)
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			matcher.patterns = append(matcher.patterns, line)
			logger.Trace("Added gitignore pattern", "pattern", line)
		}
	}

	logger.Info("Gitignore patterns loaded", "patterns", len(matcher.patterns))
	return matcher, scanner.Err()
}

func (m *Matcher) ShouldIgnore(path string) bool {
	if m == nil || len(m.patterns) == 0 {
		return false
	}

	relPath, err := filepath.Rel(m.basePath, path)
	if err != nil {
		logger.Debug("Cannot get relative path", "base", m.basePath, "path", path, "error", err)
		return false
	}

	for _, pattern := range m.patterns {
		if matched, _ := filepath.Match(pattern, filepath.Base(relPath)); matched {
			logger.Trace("File matched gitignore pattern (basename)", "path", relPath, "pattern", pattern)
			return true
		}
		if matched, _ := filepath.Match(pattern, relPath); matched {
			logger.Trace("File matched gitignore pattern (full path)", "path", relPath, "pattern", pattern)
			return true
		}
		if strings.Contains(relPath, pattern) {
			logger.Trace("File matched gitignore pattern (substring)", "path", relPath, "pattern", pattern)
			return true
		}
	}

	return false
}
