package concurrent

import (
	"maps"
	"sync"
	"time"
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
	CharMap      map[rune]int
	SequenceMap2 map[uint16]uint32
	SequenceMap3 map[uint32]uint32
	FileCount    int
	CharCount    int
}

type Worker struct {
	fileCount int
}

type WorkerPool struct {
	workerCount int
	jobs        chan FileJob
	results     chan CharCountResult
	done        chan bool
	wg          sync.WaitGroup
	workers     []*Worker
}

type ResultTiming struct {
	Values map[string]time.Duration
}

type ResultCollector struct {
	totalCharMap      map[rune]int
	totalSequenceMap2 map[uint16]uint32
	totalSequenceMap3 map[uint32]uint32
	totalFiles        int
	totalChars        int
	filesFound        int
	filesIgnored      int
	mu                sync.RWMutex
	timing            ResultTiming
}

func NewResultCollector() *ResultCollector {
	return &ResultCollector{
		totalCharMap:      make(map[rune]int),
		totalSequenceMap2: make(map[uint16]uint32),
		totalSequenceMap3: make(map[uint32]uint32),
		totalFiles:        0,
		totalChars:        0,
		filesFound:        0,
		filesIgnored:      0,
		timing: ResultTiming{
			Values: map[string]time.Duration{},
		},
	}
}

func (rc *ResultCollector) AddResult(result CharCountResult) {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	startAdding := time.Now()

	for char, count := range result.CharMap {
		rc.totalCharMap[char] += count
	}
	for seq, count := range result.SequenceMap2 {
		rc.totalSequenceMap2[seq] += count
	}
	for seq, count := range result.SequenceMap3 {
		rc.totalSequenceMap3[seq] += count
	}

	rc.totalFiles += result.FileCount
	rc.totalChars += result.CharCount

	rc.timing.Values["AddResult"] = rc.timing.Values["AddResult"] + time.Since(startAdding)

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

func (rc *ResultCollector) GetResults() (
	map[rune]int,
	map[uint16]uint32,
	map[uint32]uint32,
	int,
	int,
	int,
	int,
	ResultTiming,
) {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	// Create copies to avoid data races
	charMapCopy := make(map[rune]int)
	sequenceMap2Copy := make(map[uint16]uint32)
	sequenceMap3Copy := make(map[uint32]uint32)
	maps.Copy(charMapCopy, rc.totalCharMap)
	maps.Copy(sequenceMap2Copy, rc.totalSequenceMap2)
	maps.Copy(sequenceMap3Copy, rc.totalSequenceMap3)

	return charMapCopy,
		sequenceMap2Copy,
		sequenceMap3Copy,
		rc.totalFiles,
		rc.totalChars,
		rc.filesFound,
		rc.filesIgnored,
		rc.timing
}
