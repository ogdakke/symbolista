package concurrent

import (
	"maps"
	"sync"
)

type FileJob struct {
	Path      string
	Content   []byte
	AsciiOnly bool
}

type ProgressCallback func(filesFound, filesProcessed int)

type CharCountResult struct {
	CharMap   map[rune]int
	FileCount int
	CharCount int
}

type WorkerPool struct {
	workerCount int
	jobs        chan FileJob
	results     chan CharCountResult
	done        chan bool
	wg          sync.WaitGroup
}

type ResultCollector struct {
	totalCharMap map[rune]int
	totalFiles   int
	totalChars   int
	filesFound   int
	filesIgnored int
	mu           sync.RWMutex
}

func NewResultCollector() *ResultCollector {
	return &ResultCollector{
		totalCharMap: make(map[rune]int),
		totalFiles:   0,
		totalChars:   0,
		filesFound:   0,
		filesIgnored: 0,
	}
}

func (rc *ResultCollector) AddResult(result CharCountResult) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	for char, count := range result.CharMap {
		rc.totalCharMap[char] += count
	}
	rc.totalFiles += result.FileCount
	rc.totalChars += result.CharCount
}

func (rc *ResultCollector) IncrementFound() {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.filesFound++
}

func (rc *ResultCollector) IncrementIgnored() {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.filesIgnored++
}

func (rc *ResultCollector) GetResults() (map[rune]int, int, int, int, int) {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	// Create a copy to avoid data races
	charMapCopy := make(map[rune]int)
	maps.Copy(charMapCopy, rc.totalCharMap)

	return charMapCopy, rc.totalFiles, rc.totalChars, rc.filesFound, rc.filesIgnored
}
