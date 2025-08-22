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

	// Clean the relative path to use forward slashes consistently
	relPath = filepath.ToSlash(relPath)

	for _, pattern := range m.patterns {
		if m.matchesPattern(relPath, pattern) {
			logger.Trace("File matched gitignore pattern", "path", relPath, "pattern", pattern)
			return true
		}
	}

	return false
}

func (m *Matcher) matchesPattern(relPath, pattern string) bool {
	// Clean the pattern to use forward slashes
	pattern = filepath.ToSlash(pattern)

	// Handle directory patterns (ending with /)
	if strings.HasSuffix(pattern, "/") {
		// Directory patterns match the directory and everything inside it
		dirPattern := strings.TrimSuffix(pattern, "/")
		if relPath == dirPattern || strings.HasPrefix(relPath, dirPattern+"/") {
			return true
		}
	}

	// Handle patterns starting with /
	if strings.HasPrefix(pattern, "/") {
		pattern = strings.TrimPrefix(pattern, "/")
		// Root-anchored pattern - match from root only
		if matched, _ := filepath.Match(pattern, relPath); matched {
			return true
		}
		// For directory traversal, also check if any parent directory matches
		parts := strings.Split(relPath, "/")
		for i := range parts {
			partialPath := strings.Join(parts[:i+1], "/")
			if matched, _ := filepath.Match(pattern, partialPath); matched {
				return true
			}
		}
	} else {
		// Non-anchored pattern - can match at any level
		// Check exact match
		if matched, _ := filepath.Match(pattern, relPath); matched {
			return true
		}
		// Check basename match
		if matched, _ := filepath.Match(pattern, filepath.Base(relPath)); matched {
			return true
		}
		// Check if any directory component matches
		parts := strings.Split(relPath, "/")
		for _, part := range parts {
			if matched, _ := filepath.Match(pattern, part); matched {
				return true
			}
		}
		// Check if the pattern matches any sub-path
		for i := range parts {
			partialPath := strings.Join(parts[i:], "/")
			if matched, _ := filepath.Match(pattern, partialPath); matched {
				return true
			}
		}
	}

	return false
}
