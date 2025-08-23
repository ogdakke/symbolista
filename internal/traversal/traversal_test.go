package traversal

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ogdakke/symbolista/internal/ignorer"
)

func TestWalkDirectoryBasic(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "traversal_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testFile1 := filepath.Join(tempDir, "test1.txt")
	err = os.WriteFile(testFile1, []byte("hello world"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	testFile2 := filepath.Join(tempDir, "test2.txt")
	err = os.WriteFile(testFile2, []byte("goodbye"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	var processedFiles []string
	var processedContent []string

	processor := func(path string, content []byte) error {
		processedFiles = append(processedFiles, path)
		processedContent = append(processedContent, string(content))
		return nil
	}

	err = WalkDirectory(tempDir, nil, processor)
	if err != nil {
		t.Fatalf("WalkDirectory failed: %v", err)
	}

	if len(processedFiles) != 2 {
		t.Errorf("Expected 2 processed files, got %d", len(processedFiles))
	}

	foundTest1 := false
	foundTest2 := false
	for i, path := range processedFiles {
		if strings.HasSuffix(path, "test1.txt") {
			foundTest1 = true
			if processedContent[i] != "hello world" {
				t.Errorf("Wrong content for test1.txt: %s", processedContent[i])
			}
		}
		if strings.HasSuffix(path, "test2.txt") {
			foundTest2 = true
			if processedContent[i] != "goodbye" {
				t.Errorf("Wrong content for test2.txt: %s", processedContent[i])
			}
		}
	}

	if !foundTest1 || !foundTest2 {
		t.Error("Not all test files were processed")
	}
}

func TestWalkDirectoryWithGitignore(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "traversal_gitignore_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testFile1 := filepath.Join(tempDir, "include.txt")
	err = os.WriteFile(testFile1, []byte("include me"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	testFile2 := filepath.Join(tempDir, "ignore.log")
	err = os.WriteFile(testFile2, []byte("ignore me"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	gitignoreFile := filepath.Join(tempDir, ".gitignore")
	err = os.WriteFile(gitignoreFile, []byte("*.log\n"), 0644)
	if err != nil {
		t.Fatalf("Failed to create .gitignore: %v", err)
	}

	matcher, err := ignorer.NewMatcher(tempDir, true)
	if err != nil {
		t.Fatalf("Failed to create gitignore matcher: %v", err)
	}

	var processedFiles []string

	processor := func(path string, content []byte) error {
		processedFiles = append(processedFiles, path)
		return nil
	}

	err = WalkDirectory(tempDir, matcher, processor)
	if err != nil {
		t.Fatalf("WalkDirectory failed: %v", err)
	}

	if len(processedFiles) != 2 {
		t.Errorf("Expected 2 processed files (.gitignore and include.txt), got %d", len(processedFiles))
	}

	foundInclude := false
	for _, path := range processedFiles {
		if strings.HasSuffix(path, "include.txt") {
			foundInclude = true
			break
		}
	}
	if !foundInclude {
		t.Error("Expected include.txt to be processed")
	}
}

func TestWalkDirectoryWithNestedStructure(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "traversal_nested_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	subDir := filepath.Join(tempDir, "subdir")
	err = os.Mkdir(subDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create subdir: %v", err)
	}

	deepDir := filepath.Join(subDir, "deep")
	err = os.Mkdir(deepDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create deep dir: %v", err)
	}

	rootFile := filepath.Join(tempDir, "root.txt")
	err = os.WriteFile(rootFile, []byte("root"), 0644)
	if err != nil {
		t.Fatalf("Failed to create root file: %v", err)
	}

	subFile := filepath.Join(subDir, "sub.txt")
	err = os.WriteFile(subFile, []byte("sub"), 0644)
	if err != nil {
		t.Fatalf("Failed to create sub file: %v", err)
	}

	deepFile := filepath.Join(deepDir, "deep.txt")
	err = os.WriteFile(deepFile, []byte("deep"), 0644)
	if err != nil {
		t.Fatalf("Failed to create deep file: %v", err)
	}

	var processedFiles []string

	processor := func(path string, content []byte) error {
		processedFiles = append(processedFiles, path)
		return nil
	}

	err = WalkDirectory(tempDir, nil, processor)
	if err != nil {
		t.Fatalf("WalkDirectory failed: %v", err)
	}

	if len(processedFiles) != 3 {
		t.Errorf("Expected 3 processed files, got %d", len(processedFiles))
	}

	foundRoot := false
	foundSub := false
	foundDeep := false
	for _, path := range processedFiles {
		if strings.HasSuffix(path, "root.txt") {
			foundRoot = true
		}
		if strings.HasSuffix(path, "sub.txt") {
			foundSub = true
		}
		if strings.HasSuffix(path, "deep.txt") {
			foundDeep = true
		}
	}

	if !foundRoot || !foundSub || !foundDeep {
		t.Error("Not all nested files were processed")
	}
}

func TestWalkDirectoryWithHierarchicalGitignore(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "traversal_hierarchical_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create nested directory structure
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

	// Create project .gitignore (ignores *.tmp)
	projectGitignore := filepath.Join(projectDir, ".gitignore")
	err = os.WriteFile(projectGitignore, []byte("*.tmp\n"), 0644)
	if err != nil {
		t.Fatalf("Failed to create project .gitignore: %v", err)
	}

	// Create test files
	rootFile := filepath.Join(tempDir, "root.txt")
	err = os.WriteFile(rootFile, []byte("root"), 0644)
	if err != nil {
		t.Fatalf("Failed to create root file: %v", err)
	}

	rootLog := filepath.Join(tempDir, "root.log")
	err = os.WriteFile(rootLog, []byte("root log"), 0644)
	if err != nil {
		t.Fatalf("Failed to create root log: %v", err)
	}

	projectFile := filepath.Join(projectDir, "project.txt")
	err = os.WriteFile(projectFile, []byte("project"), 0644)
	if err != nil {
		t.Fatalf("Failed to create project file: %v", err)
	}

	projectLog := filepath.Join(projectDir, "project.log")
	err = os.WriteFile(projectLog, []byte("project log"), 0644)
	if err != nil {
		t.Fatalf("Failed to create project log: %v", err)
	}

	projectTmp := filepath.Join(projectDir, "project.tmp")
	err = os.WriteFile(projectTmp, []byte("project tmp"), 0644)
	if err != nil {
		t.Fatalf("Failed to create project tmp: %v", err)
	}

	srcFile := filepath.Join(srcDir, "src.txt")
	err = os.WriteFile(srcFile, []byte("src"), 0644)
	if err != nil {
		t.Fatalf("Failed to create src file: %v", err)
	}

	srcLog := filepath.Join(srcDir, "src.log")
	err = os.WriteFile(srcLog, []byte("src log"), 0644)
	if err != nil {
		t.Fatalf("Failed to create src log: %v", err)
	}

	srcTmp := filepath.Join(srcDir, "src.tmp")
	err = os.WriteFile(srcTmp, []byte("src tmp"), 0644)
	if err != nil {
		t.Fatalf("Failed to create src tmp: %v", err)
	}

	matcher, err := ignorer.NewMatcher(tempDir, true)
	if err != nil {
		t.Fatalf("Failed to create gitignore matcher: %v", err)
	}

	var processedFiles []string

	processor := func(path string, content []byte) error {
		processedFiles = append(processedFiles, path)
		return nil
	}

	err = WalkDirectory(tempDir, matcher, processor)
	if err != nil {
		t.Fatalf("WalkDirectory failed: %v", err)
	}

	processedSet := make(map[string]bool)
	for _, path := range processedFiles {
		processedSet[filepath.Base(path)] = true
	}

	expectedProcessed := []string{"root.txt", "project.txt", "src.txt"}
	for _, expected := range expectedProcessed {
		if !processedSet[expected] {
			t.Errorf("Expected file %s to be processed", expected)
		}
	}

	expectedIgnored := []string{"root.log", "project.log", "project.tmp", "src.log", "src.tmp"}
	for _, expected := range expectedIgnored {
		if processedSet[expected] {
			t.Errorf("Expected file %s to be ignored", expected)
		}
	}
}

func TestWalkDirectoryIgnoresDirectories(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "traversal_ignore_dir_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create directory structure
	includeDir := filepath.Join(tempDir, "include")
	ignoreDir := filepath.Join(tempDir, "node_modules")
	err = os.MkdirAll(includeDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create include dir: %v", err)
	}
	err = os.MkdirAll(ignoreDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create ignore dir: %v", err)
	}

	// Create files in both directories
	includeFile := filepath.Join(includeDir, "include.txt")
	err = os.WriteFile(includeFile, []byte("include"), 0644)
	if err != nil {
		t.Fatalf("Failed to create include file: %v", err)
	}

	ignoreFile := filepath.Join(ignoreDir, "ignore.txt")
	err = os.WriteFile(ignoreFile, []byte("ignore"), 0644)
	if err != nil {
		t.Fatalf("Failed to create ignore file: %v", err)
	}

	// Create .gitignore that ignores node_modules directory
	gitignoreFile := filepath.Join(tempDir, ".gitignore")
	err = os.WriteFile(gitignoreFile, []byte("node_modules/\n"), 0644)
	if err != nil {
		t.Fatalf("Failed to create .gitignore: %v", err)
	}

	// Create gitignore matcher - include dotfiles so we can test processing .gitignore
	matcher, err := ignorer.NewMatcher(tempDir, true)
	if err != nil {
		t.Fatalf("Failed to create gitignore matcher: %v", err)
	}

	// Track processed files
	var processedFiles []string

	processor := func(path string, content []byte) error {
		processedFiles = append(processedFiles, path)
		return nil
	}

	err = WalkDirectory(tempDir, matcher, processor)
	if err != nil {
		t.Fatalf("WalkDirectory failed: %v", err)
	}

	// Should have processed include.txt and .gitignore file (node_modules should be ignored)
	if len(processedFiles) != 2 {
		t.Errorf("Expected 2 processed files (.gitignore and include.txt), got %d", len(processedFiles))
	}

	// Check that include.txt was processed
	foundInclude := false
	for _, path := range processedFiles {
		if strings.HasSuffix(path, "include.txt") {
			foundInclude = true
			break
		}
	}
	if !foundInclude {
		t.Error("Expected include.txt to be processed")
	}
}

func TestWalkDirectorySkipsBinaryFiles(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "traversal_binary_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a text file
	textFile := filepath.Join(tempDir, "text.txt")
	err = os.WriteFile(textFile, []byte("hello world"), 0644)
	if err != nil {
		t.Fatalf("Failed to create text file: %v", err)
	}

	// Create a binary file (with invalid UTF-8)
	binaryFile := filepath.Join(tempDir, "binary.bin")
	binaryData := []byte{0xFF, 0xFE, 0x00, 0x01, 0x80, 0x90}
	err = os.WriteFile(binaryFile, binaryData, 0644)
	if err != nil {
		t.Fatalf("Failed to create binary file: %v", err)
	}

	// Track processed files
	var processedFiles []string

	processor := func(path string, content []byte) error {
		processedFiles = append(processedFiles, path)
		return nil
	}

	err = WalkDirectory(tempDir, nil, processor)
	if err != nil {
		t.Fatalf("WalkDirectory failed: %v", err)
	}

	// Should have processed only the text file
	if len(processedFiles) != 1 {
		t.Errorf("Expected 1 processed file, got %d", len(processedFiles))
	}

	if !strings.HasSuffix(processedFiles[0], "text.txt") {
		t.Errorf("Expected text.txt to be processed, got %s", processedFiles[0])
	}
}

func TestWalkDirectoryHandlesErrors(t *testing.T) {
	// Test with non-existent directory
	err := WalkDirectory("/nonexistent/directory", nil, func(path string, content []byte) error {
		return nil
	})

	if err == nil {
		t.Error("Expected error for non-existent directory")
	}
}

func TestProcessorError(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "traversal_error_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test file
	testFile := filepath.Join(tempDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create a processor that returns an error
	processor := func(path string, content []byte) error {
		return os.ErrPermission
	}

	err = WalkDirectory(tempDir, nil, processor)
	if err == nil {
		t.Error("Expected error from processor")
	}
}
