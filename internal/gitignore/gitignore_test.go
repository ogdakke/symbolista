package gitignore

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewMatcher(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "gitignore_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Test with no .gitignore file
	matcher, err := NewMatcher(tempDir)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if matcher == nil {
		t.Fatal("Expected matcher, got nil")
	}
	if len(matcher.patterns) != 0 {
		t.Errorf("Expected no patterns, got %d", len(matcher.patterns))
	}

	// Create a .gitignore file
	gitignorePath := filepath.Join(tempDir, ".gitignore")
	gitignoreContent := `# This is a comment
*.log
node_modules/
build/
# Another comment

*.tmp`

	err = os.WriteFile(gitignorePath, []byte(gitignoreContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create .gitignore: %v", err)
	}

	// Test with .gitignore file
	matcher, err = NewMatcher(tempDir)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if matcher == nil {
		t.Fatal("Expected matcher, got nil")
	}

	expectedPatterns := []string{"*.log", "node_modules/", "build/", "*.tmp"}
	if len(matcher.patterns) != len(expectedPatterns) {
		t.Errorf("Expected %d patterns, got %d", len(expectedPatterns), len(matcher.patterns))
	}

	for i, expected := range expectedPatterns {
		if i >= len(matcher.patterns) || matcher.patterns[i] != expected {
			t.Errorf("Expected pattern %q at index %d, got %q", expected, i, matcher.patterns[i])
		}
	}
}

func TestLoadGitignoreForDirectory(t *testing.T) {
	// Create a temporary directory structure
	tempDir, err := os.MkdirTemp("", "gitignore_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	subDir := filepath.Join(tempDir, "subdir")
	err = os.Mkdir(subDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create subdir: %v", err)
	}

	// Create root .gitignore
	rootGitignore := filepath.Join(tempDir, ".gitignore")
	err = os.WriteFile(rootGitignore, []byte("*.log\n"), 0644)
	if err != nil {
		t.Fatalf("Failed to create root .gitignore: %v", err)
	}

	// Create sub .gitignore
	subGitignore := filepath.Join(subDir, ".gitignore")
	err = os.WriteFile(subGitignore, []byte("*.tmp\nnode_modules/\n"), 0644)
	if err != nil {
		t.Fatalf("Failed to create sub .gitignore: %v", err)
	}

	matcher, err := NewMatcher(tempDir)
	if err != nil {
		t.Fatalf("Failed to create matcher: %v", err)
	}

	// Load gitignore for subdirectory
	err = matcher.LoadGitignoreForDirectory(subDir)
	if err != nil {
		t.Errorf("Expected no error loading subdirectory gitignore, got %v", err)
	}

	// Check that both gitignore files are loaded
	if len(matcher.matchers) != 2 {
		t.Errorf("Expected 2 matcher entries, got %d", len(matcher.matchers))
	}

	// Check root patterns
	rootPatterns, exists := matcher.matchers[tempDir]
	if !exists {
		t.Error("Expected root gitignore patterns to be loaded")
	} else if len(rootPatterns) != 1 || rootPatterns[0] != "*.log" {
		t.Errorf("Expected root pattern [*.log], got %v", rootPatterns)
	}

	// Check sub patterns
	subPatterns, exists := matcher.matchers[subDir]
	if !exists {
		t.Error("Expected sub gitignore patterns to be loaded")
	} else if len(subPatterns) != 2 {
		t.Errorf("Expected 2 sub patterns, got %d", len(subPatterns))
	}
}

func TestShouldIgnore(t *testing.T) {
	// Create a temporary directory structure
	tempDir, err := os.MkdirTemp("", "gitignore_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	subDir := filepath.Join(tempDir, "project")
	err = os.Mkdir(subDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create subdir: %v", err)
	}

	// Create root .gitignore
	rootGitignore := filepath.Join(tempDir, ".gitignore")
	err = os.WriteFile(rootGitignore, []byte("*.log\n"), 0644)
	if err != nil {
		t.Fatalf("Failed to create root .gitignore: %v", err)
	}

	// Create sub .gitignore
	subGitignore := filepath.Join(subDir, ".gitignore")
	err = os.WriteFile(subGitignore, []byte("*.tmp\nnode_modules/\n"), 0644)
	if err != nil {
		t.Fatalf("Failed to create sub .gitignore: %v", err)
	}

	matcher, err := NewMatcher(tempDir)
	if err != nil {
		t.Fatalf("Failed to create matcher: %v", err)
	}

	// Load gitignore for subdirectory
	err = matcher.LoadGitignoreForDirectory(subDir)
	if err != nil {
		t.Fatalf("Failed to load subdirectory gitignore: %v", err)
	}

	tests := []struct {
		path     string
		expected bool
		desc     string
	}{
		{filepath.Join(tempDir, "test.log"), true, "root .log file should be ignored"},
		{filepath.Join(subDir, "test.log"), true, "sub .log file should be ignored (inherited from root)"},
		{filepath.Join(subDir, "test.tmp"), true, "sub .tmp file should be ignored (local rule)"},
		{filepath.Join(subDir, "node_modules", "package.json"), true, "files in node_modules should be ignored"},
		{filepath.Join(subDir, "src", "main.go"), false, "regular files should not be ignored"},
		{filepath.Join(tempDir, "test.txt"), false, "regular files in root should not be ignored"},
		{filepath.Join(subDir, "test.js"), false, "regular js files should not be ignored"},
	}

	for _, test := range tests {
		result := matcher.ShouldIgnore(test.path)
		if result != test.expected {
			t.Errorf("%s: expected %v, got %v", test.desc, test.expected, result)
		}
	}
}

func TestMatchesPattern(t *testing.T) {
	matcher := &Matcher{}

	tests := []struct {
		relPath  string
		pattern  string
		expected bool
		desc     string
	}{
		{"test.log", "*.log", true, "simple glob pattern"},
		{"src/test.log", "*.log", true, "glob pattern in subdirectory"},
		{"test.txt", "*.log", false, "non-matching glob pattern"},
		{"node_modules/package.json", "node_modules/", true, "directory pattern"},
		{"node_modules", "node_modules/", true, "directory pattern exact match"},
		{"src/node_modules/test.js", "node_modules/", true, "directory pattern in subdirectory"},
		{"src/test.js", "/src/*.js", true, "root-anchored pattern"},
		{"deep/src/test.js", "/src/*.js", false, "root-anchored pattern in wrong location"},
		{"build/output.txt", "build", true, "directory name pattern"},
		{"src/build/output.txt", "build", true, "directory name pattern in subdirectory"},
		{"rebuild.txt", "build", false, "partial match should not work"},
	}

	for _, test := range tests {
		result := matcher.matchesPattern(test.relPath, test.pattern)
		if result != test.expected {
			t.Errorf("%s: pattern %q on path %q expected %v, got %v",
				test.desc, test.pattern, test.relPath, test.expected, result)
		}
	}
}

func TestShouldIgnoreWithNilMatcher(t *testing.T) {
	var matcher *Matcher
	result := matcher.ShouldIgnore("/some/path")
	if result != false {
		t.Errorf("Expected false for nil matcher, got %v", result)
	}
}

func TestHierarchicalGitignore(t *testing.T) {
	// Create a complex directory structure
	tempDir, err := os.MkdirTemp("", "gitignore_hierarchy_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create directory structure: root/project/src/
	projectDir := filepath.Join(tempDir, "project")
	srcDir := filepath.Join(projectDir, "src")
	err = os.MkdirAll(srcDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create directory structure: %v", err)
	}

	// Create root .gitignore (ignores *.log)
	rootGitignore := filepath.Join(tempDir, ".gitignore")
	err = os.WriteFile(rootGitignore, []byte("*.log\n"), 0644)
	if err != nil {
		t.Fatalf("Failed to create root .gitignore: %v", err)
	}

	// Create project .gitignore (ignores *.tmp and build/)
	projectGitignore := filepath.Join(projectDir, ".gitignore")
	err = os.WriteFile(projectGitignore, []byte("*.tmp\nbuild/\n"), 0644)
	if err != nil {
		t.Fatalf("Failed to create project .gitignore: %v", err)
	}

	// Create src .gitignore (ignores *.bak)
	srcGitignore := filepath.Join(srcDir, ".gitignore")
	err = os.WriteFile(srcGitignore, []byte("*.bak\n"), 0644)
	if err != nil {
		t.Fatalf("Failed to create src .gitignore: %v", err)
	}

	matcher, err := NewMatcher(tempDir)
	if err != nil {
		t.Fatalf("Failed to create matcher: %v", err)
	}

	// Load gitignore files for each directory
	err = matcher.LoadGitignoreForDirectory(projectDir)
	if err != nil {
		t.Fatalf("Failed to load project gitignore: %v", err)
	}

	err = matcher.LoadGitignoreForDirectory(srcDir)
	if err != nil {
		t.Fatalf("Failed to load src gitignore: %v", err)
	}

	tests := []struct {
		path     string
		expected bool
		desc     string
	}{
		// Files that should be ignored by root .gitignore
		{filepath.Join(tempDir, "debug.log"), true, "root level .log file"},
		{filepath.Join(projectDir, "app.log"), true, "project level .log file (inherited from root)"},
		{filepath.Join(srcDir, "error.log"), true, "src level .log file (inherited from root)"},

		// Files that should be ignored by project .gitignore
		{filepath.Join(projectDir, "temp.tmp"), true, "project level .tmp file"},
		{filepath.Join(srcDir, "cache.tmp"), true, "src level .tmp file (inherited from project)"},
		{filepath.Join(projectDir, "build", "output.txt"), true, "files in build directory"},

		// Files that should be ignored by src .gitignore
		{filepath.Join(srcDir, "backup.bak"), true, "src level .bak file"},

		// Files that should NOT be ignored
		{filepath.Join(tempDir, "readme.txt"), false, "root level regular file"},
		{filepath.Join(projectDir, "main.go"), false, "project level regular file"},
		{filepath.Join(srcDir, "utils.go"), false, "src level regular file"},
		{filepath.Join(tempDir, "temp.tmp"), false, ".tmp file at root (not covered by project gitignore)"},
		{filepath.Join(tempDir, "backup.bak"), false, ".bak file at root (not covered by src gitignore)"},
	}

	for _, test := range tests {
		result := matcher.ShouldIgnore(test.path)
		if result != test.expected {
			t.Errorf("%s: path %q expected %v, got %v", test.desc, test.path, test.expected, result)
		}
	}
}
