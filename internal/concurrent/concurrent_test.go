package concurrent

import (
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"

	"github.com/ogdakke/symbolista/internal/gitignore"
)

func TestNewWorkerPool(t *testing.T) {
	tests := []struct {
		name            string
		workerCount     int
		jobBufferSize   int
		expectedWorkers int
	}{
		{"Default worker count", 0, 10, runtime.NumCPU()},
		{"Specific worker count", 4, 10, 4},
		{"Negative worker count", -1, 10, runtime.NumCPU()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool := NewWorkerPool(tt.workerCount, tt.jobBufferSize)
			if pool.workerCount != tt.expectedWorkers {
				t.Errorf("Expected %d workers, got %d", tt.expectedWorkers, pool.workerCount)
			}
		})
	}
}

func TestResultCollector(t *testing.T) {
	collector := NewResultCollector()

	// Test empty collector
	charMap, fileCount, totalChars := collector.GetResults()
	if len(charMap) != 0 || fileCount != 0 || totalChars != 0 {
		t.Errorf("Expected empty results, got charMap=%d, fileCount=%d, totalChars=%d",
			len(charMap), fileCount, totalChars)
	}

	// Add some results
	result1 := CharCountResult{
		CharMap:   map[rune]int{'a': 5, 'b': 3},
		FileCount: 1,
		CharCount: 8,
	}
	result2 := CharCountResult{
		CharMap:   map[rune]int{'a': 2, 'c': 4},
		FileCount: 1,
		CharCount: 6,
	}

	collector.AddResult(result1)
	collector.AddResult(result2)

	charMap, fileCount, totalChars = collector.GetResults()

	if fileCount != 2 {
		t.Errorf("Expected 2 files, got %d", fileCount)
	}
	if totalChars != 14 {
		t.Errorf("Expected 14 total chars, got %d", totalChars)
	}
	if charMap['a'] != 7 {
		t.Errorf("Expected 'a' count 7, got %d", charMap['a'])
	}
	if charMap['b'] != 3 {
		t.Errorf("Expected 'b' count 3, got %d", charMap['b'])
	}
	if charMap['c'] != 4 {
		t.Errorf("Expected 'c' count 4, got %d", charMap['c'])
	}
}

func TestConcurrentResultCollector(t *testing.T) {
	collector := NewResultCollector()

	// Test thread safety with concurrent access
	var wg sync.WaitGroup
	numGoroutines := 10
	resultsPerGoroutine := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < resultsPerGoroutine; j++ {
				result := CharCountResult{
					CharMap:   map[rune]int{rune('a' + id): 1},
					FileCount: 1,
					CharCount: 1,
				}
				collector.AddResult(result)
			}
		}(i)
	}

	wg.Wait()

	charMap, fileCount, totalChars := collector.GetResults()

	expectedFiles := numGoroutines * resultsPerGoroutine
	if fileCount != expectedFiles {
		t.Errorf("Expected %d files, got %d", expectedFiles, fileCount)
	}
	if totalChars != expectedFiles {
		t.Errorf("Expected %d total chars, got %d", expectedFiles, totalChars)
	}
	if len(charMap) != numGoroutines {
		t.Errorf("Expected %d unique chars, got %d", numGoroutines, len(charMap))
	}
}

func TestWorkerPool(t *testing.T) {
	// Create a temporary test file
	tmpDir, err := os.MkdirTemp("", "concurrent_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "Hello, World! This is a test file."
	err = os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	pool := NewWorkerPool(2, 5)
	pool.Start()

	// Send a job
	job := FileJob{
		Path:    testFile,
		Content: []byte(testContent),
	}
	pool.AddJob(job)
	pool.CloseJobs()

	// Collect results
	var results []CharCountResult
	for result := range pool.Results() {
		results = append(results, result)
	}

	<-pool.Done()

	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}

	result := results[0]
	if result.FileCount != 1 {
		t.Errorf("Expected file count 1, got %d", result.FileCount)
	}
	if result.CharCount != len(testContent) {
		t.Errorf("Expected char count %d, got %d", len(testContent), result.CharCount)
	}
}

func TestDiscoverFiles(t *testing.T) {
	// Create a temporary directory structure
	tmpDir, err := os.MkdirTemp("", "discover_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files
	testFile1 := filepath.Join(tmpDir, "test1.txt")
	testFile2 := filepath.Join(tmpDir, "test2.txt")

	err = os.WriteFile(testFile1, []byte("content1"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(testFile2, []byte("content2"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Create gitignore matcher
	matcher, err := gitignore.NewMatcher(tmpDir, false)
	if err != nil {
		t.Fatal(err)
	}

	jobChan := make(chan FileJob, 10)
	var discoveryError error

	go DiscoverFiles(tmpDir, matcher, jobChan, true, func(err error) {
		discoveryError = err
	})

	// Collect discovered jobs
	var jobs []FileJob
	for job := range jobChan {
		jobs = append(jobs, job)
	}

	if discoveryError != nil {
		t.Errorf("Discovery error: %v", discoveryError)
	}

	if len(jobs) != 2 {
		t.Errorf("Expected 2 jobs, got %d", len(jobs))
	}

	// Verify job contents
	contentMap := make(map[string]string)
	for _, job := range jobs {
		contentMap[job.Path] = string(job.Content)
	}

	if contentMap[testFile1] != "content1" {
		t.Errorf("Expected content1, got %s", contentMap[testFile1])
	}
	if contentMap[testFile2] != "content2" {
		t.Errorf("Expected content2, got %s", contentMap[testFile2])
	}
}
