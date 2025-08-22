package gitignore

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ogdakke/symbolista/internal/logger"
)

type Matcher struct {
	patterns []string
	basePath string
	// Stack of gitignore matchers for nested directories
	matchers map[string][]string
}

func NewMatcher(basePath string) (*Matcher, error) {
	matcher := &Matcher{
		basePath: basePath,
		matchers: make(map[string][]string),
	}

	// Load root gitignore if it exists
	if err := matcher.loadGitignoreForDir(basePath); err != nil {
		return nil, err
	}

	return matcher, nil
}

func (m *Matcher) loadGitignoreForDir(dirPath string) error {
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
		// Also add to legacy patterns for root directory compatibility
		if dirPath == m.basePath {
			m.patterns = patterns
		}
		logger.Info("Gitignore patterns loaded", "patterns", len(patterns), "dir", dirPath)
	}

	return scanner.Err()
}

func (m *Matcher) LoadGitignoreForDirectory(dirPath string) error {
	return m.loadGitignoreForDir(dirPath)
}

func (m *Matcher) ShouldIgnore(path string) bool {
	if m == nil {
		return false
	}

	start := time.Now()

	// Check all gitignore files from root to the directory containing this path
	currentDir := filepath.Dir(path)
	for {
		// Check if currentDir is within our base path
		relDir, err := filepath.Rel(m.basePath, currentDir)
		if err != nil || strings.HasPrefix(relDir, "..") {
			break
		}

		// Check patterns from this directory's gitignore
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

		// Move up one directory
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
		// Also check if the directory pattern matches any component in the path
		parts := strings.Split(relPath, "/")
		for i, part := range parts {
			if part == dirPattern {
				// Found the directory, now check if we're inside it or it's the exact match
				if i == len(parts)-1 || len(parts) > i+1 {
					return true
				}
			}
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
