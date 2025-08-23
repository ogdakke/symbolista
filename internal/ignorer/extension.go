package ignorer

import (
	"path/filepath"

	"github.com/ogdakke/symbolista/internal/logger"
)

type ExtensionIgnorer struct {
	ignoredExtensions map[string]bool
}

func NewExtensionIgnorer() *ExtensionIgnorer {
	ignorer := &ExtensionIgnorer{
		ignoredExtensions: make(map[string]bool),
	}

	ignorer.addDefaultIgnoredExtensions()
	return ignorer
}

func (e *ExtensionIgnorer) addDefaultIgnoredExtensions() {
	defaultIgnored := []string{
		".svg",
	}

	for _, ext := range defaultIgnored {
		e.ignoredExtensions[ext] = true
	}
}

func (e *ExtensionIgnorer) ShouldIgnore(path string) bool {
	ext := filepath.Ext(path)
	if ext != "" && e.ignoredExtensions[ext] {
		logger.Trace("Ignoring file by extension", "path", path, "extension", ext)
		return true
	}
	return false
}

func (e *ExtensionIgnorer) AddExtension(ext string) {
	if ext != "" {
		e.ignoredExtensions[ext] = true
	}
}

func (e *ExtensionIgnorer) RemoveExtension(ext string) {
	delete(e.ignoredExtensions, ext)
}
