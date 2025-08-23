package ignorer

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ogdakke/symbolista/internal/logger"
)

type GitignoreMatcher struct {
	patterns []string
	basePath string
	// Stack of gitignore matchers for nested directories
	matchers map[string][]string
}

func NewGitignoreMatcher(basePath string) (*GitignoreMatcher, error) {
	matcher := &GitignoreMatcher{
		basePath: basePath,
		matchers: make(map[string][]string),
	}

	if err := matcher.loadGitignoreForDir(basePath); err != nil {
		return nil, err
	}

	return matcher, nil
}

func (m *GitignoreMatcher) loadGitignoreForDir(dirPath string) error {
	gitignorePath := filepath.Join(dirPath, ".gitignore")
	if _, err := os.Stat(gitignorePath); os.IsNotExist(err) {
		logger.Debug("No .gitignore found", "path", gitignorePath)
		return nil
	}

	logger.Debug("Loading .gitignore", "path", gitignorePath)
	file, err := os.Open(gitignorePath)
	if err != nil {
		logger.Error("Cannot open .gitignore", "path", gitignorePath, "error", err)
		return err
	}
	defer file.Close()

	var patterns []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			patterns = append(patterns, line)
			logger.Trace("Added gitignore pattern", "pattern", line, "dir", dirPath)
		}
	}

	if len(patterns) > 0 {
		m.matchers[dirPath] = patterns
		if dirPath == m.basePath {
			m.patterns = patterns
		}
		logger.Info("Gitignore patterns loaded", "patterns", len(patterns), "dir", dirPath)
	}

	return scanner.Err()
}

func (m *GitignoreMatcher) LoadGitignoreForDirectory(dirPath string) error {
	return m.loadGitignoreForDir(dirPath)
}

func (m *GitignoreMatcher) ShouldIgnore(path string) bool {
	if m == nil {
		return false
	}

	start := time.Now()

	currentDir := filepath.Dir(path)
	for {
		relDir, err := filepath.Rel(m.basePath, currentDir)
		if err != nil || strings.HasPrefix(relDir, "..") {
			break
		}

		if patterns, exists := m.matchers[currentDir]; exists {
			// Get relative path from this directory's perspective
			relPath, err := filepath.Rel(currentDir, path)
			if err != nil {
				logger.Debug("Cannot get relative path", "base", currentDir, "path", path, "error", err)
			} else {
				relPath = filepath.ToSlash(relPath)
				for _, pattern := range patterns {
					if m.matchesPattern(relPath, pattern) {
						duration := time.Since(start)
						logger.Trace("File matched gitignore pattern", "path", relPath, "pattern", pattern, "gitignore_dir", currentDir, "match_duration", duration)
						return true
					}
				}
			}
		}

		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir || parentDir == "." {
			break
		}
		currentDir = parentDir
	}

	duration := time.Since(start)
	if duration > time.Microsecond*100 {
		logger.Trace("Gitignore pattern matching completed", "path", path, "duration", duration)
	}

	return false
}

func (m *GitignoreMatcher) matchesPattern(relPath, pattern string) bool {
	pattern = filepath.ToSlash(pattern)

	if strings.HasSuffix(pattern, "/") {
		dirPattern := strings.TrimSuffix(pattern, "/")
		if relPath == dirPattern || strings.HasPrefix(relPath, dirPattern+"/") {
			return true
		}

		parts := strings.Split(relPath, "/")
		for i, part := range parts {
			if part == dirPattern {
				if i == len(parts)-1 || len(parts) > i+1 {
					return true
				}
			}
		}
	}

	// Handle patterns starting with /
	if after, ok := strings.CutPrefix(pattern, "/"); ok {
		pattern = after
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
		if matched, _ := filepath.Match(pattern, relPath); matched {
			return true
		}

		if matched, _ := filepath.Match(pattern, filepath.Base(relPath)); matched {
			return true
		}

		parts := strings.Split(relPath, "/")
		for _, part := range parts {
			if matched, _ := filepath.Match(pattern, part); matched {
				return true
			}
		}

		for i := range parts {
			partialPath := strings.Join(parts[i:], "/")
			if matched, _ := filepath.Match(pattern, partialPath); matched {
				return true
			}
		}
	}

	return false
}
