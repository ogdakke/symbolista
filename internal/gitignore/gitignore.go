package gitignore

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
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
		return matcher, nil
	}

	file, err := os.Open(gitignorePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			matcher.patterns = append(matcher.patterns, line)
		}
	}

	return matcher, scanner.Err()
}

func (m *Matcher) ShouldIgnore(path string) bool {
	if m == nil || len(m.patterns) == 0 {
		return false
	}

	relPath, err := filepath.Rel(m.basePath, path)
	if err != nil {
		return false
	}

	for _, pattern := range m.patterns {
		if matched, _ := filepath.Match(pattern, filepath.Base(relPath)); matched {
			return true
		}
		if matched, _ := filepath.Match(pattern, relPath); matched {
			return true
		}
		if strings.Contains(relPath, pattern) {
			return true
		}
	}

	return false
}
