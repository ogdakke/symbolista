package concurrent

import (
	"maps"
	"sync"
)

type FileJob struct {
	Path           string
	Content        []byte
	AsciiOnly      bool
	SequenceConfig SequenceConfig
}

type SequenceConfig struct {
	Enabled   bool
	MinLength int
	MaxLength int
	Threshold int
}

type ProgressCallback func(filesFound, filesProcessed int)

type CharCountResult struct {
	CharMap     map[rune]int
	SequenceMap map[string]int
	FileCount   int
	CharCount   int
}

type WorkerPool struct {
	workerCount int
	jobs        chan FileJob
	results     chan CharCountResult
	done        chan bool
	wg          sync.WaitGroup
}

type ResultCollector struct {
	totalCharMap     map[rune]int
	totalSequenceMap map[string]int
	totalFiles       int
	totalChars       int
	filesFound       int
	filesIgnored     int
	mu               sync.RWMutex
}

func NewResultCollector() *ResultCollector {
	return &ResultCollector{
		totalCharMap:     make(map[rune]int),
		totalSequenceMap: make(map[string]int),
		totalFiles:       0,
		totalChars:       0,
		filesFound:       0,
		filesIgnored:     0,
	}
}

func (rc *ResultCollector) AddResult(result CharCountResult) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	for char, count := range result.CharMap {
		rc.totalCharMap[char] += count
	}
	for seq, count := range result.SequenceMap {
		rc.totalSequenceMap[seq] += count
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

func (rc *ResultCollector) GetResults() (map[rune]int, map[string]int, int, int, int, int) {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	// Create copies to avoid data races
	charMapCopy := make(map[rune]int)
	sequenceMapCopy := make(map[string]int)
	maps.Copy(charMapCopy, rc.totalCharMap)
	maps.Copy(sequenceMapCopy, rc.totalSequenceMap)

	return charMapCopy, sequenceMapCopy, rc.totalFiles, rc.totalChars, rc.filesFound, rc.filesIgnored
}
