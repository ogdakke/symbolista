package ignorer

import (
	"path/filepath"
	"strings"

	"github.com/ogdakke/symbolista/internal/logger"
)

type Matcher struct {
	gitignoreMatcher *GitignoreMatcher
	extensionIgnorer *ExtensionIgnorer
	includeDotfiles  bool
}

func NewMatcher(basePath string, includeDotfiles bool) (*Matcher, error) {
	gitignoreMatcher, err := NewGitignoreMatcher(basePath)
	if err != nil {
		return nil, err
	}

	extensionIgnorer := NewExtensionIgnorer()

	matcher := &Matcher{
		gitignoreMatcher: gitignoreMatcher,
		extensionIgnorer: extensionIgnorer,
		includeDotfiles:  includeDotfiles,
	}

	return matcher, nil
}

func (m *Matcher) LoadGitignoreForDirectory(dirPath string) error {
	return m.gitignoreMatcher.LoadGitignoreForDirectory(dirPath)
}

func (m *Matcher) ShouldIgnore(path string) bool {
	if m == nil {
		return false
	}

	if m.extensionIgnorer.ShouldIgnore(path) {
		return true
	}

	if !m.includeDotfiles {
		filename := filepath.Base(path)
		if strings.HasPrefix(filename, ".") && filename != "." && filename != ".." {
			logger.Trace("Ignoring dotfile", "path", path)
			return true
		}
	}

	return m.gitignoreMatcher.ShouldIgnore(path)
}
