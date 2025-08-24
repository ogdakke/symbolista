package concurrent

import (
	"runtime"
	"strings"
	"unicode"

	"github.com/ogdakke/symbolista/internal/logger"
)

func NewWorkerPool(workerCount int, jobBufferSize int) *WorkerPool {
	if workerCount <= 0 {
		workerCount = runtime.NumCPU()
	}

	return &WorkerPool{
		workerCount: workerCount,
		jobs:        make(chan FileJob, jobBufferSize),
		results:     make(chan CharCountResult, jobBufferSize),
		done:        make(chan bool),
	}
}

func (wp *WorkerPool) Start() {
	logger.Debug("Starting worker pool", "workers", wp.workerCount)

	for i := 0; i < wp.workerCount; i++ {
		wp.wg.Add(1)
		go wp.worker(i)
	}

	go func() {
		wp.wg.Wait()
		close(wp.results)
		wp.done <- true
	}()
}

func (wp *WorkerPool) AddJob(job FileJob) {
	wp.jobs <- job
}

func (wp *WorkerPool) Jobs() chan<- FileJob {
	return wp.jobs
}

func (wp *WorkerPool) CloseJobs() {
	close(wp.jobs)
}

func (wp *WorkerPool) Results() <-chan CharCountResult {
	return wp.results
}

func (wp *WorkerPool) Done() <-chan bool {
	return wp.done
}

func (wp *WorkerPool) worker(id int) {
	defer wp.wg.Done()

	logger.Trace("Worker started", "worker_id", id)

	for job := range wp.jobs {
		result := wp.processFile(job, id)
		wp.results <- result
	}

	logger.Trace("Worker finished", "worker_id", id)
}

func (wp *WorkerPool) processFile(job FileJob, workerID int) CharCountResult {
	charMap := make(map[rune]int)
	sequenceMap := make(map[string]int)
	charCount := 0

	logger.Trace("Processing file", "path", job.Path, "worker_id", workerID, "size", len(job.Content))

	content := string(job.Content)

	for _, r := range content {
		if unicode.IsGraphic(r) || unicode.IsSpace(r) {
			if job.AsciiOnly && r > 127 {
				continue
			}
			normalizedChar := []rune(strings.ToLower(string(r)))[0]
			charMap[normalizedChar]++
			charCount++
		}
	}

	if job.SequenceConfig.Enabled {
		extractSequences(content, job.AsciiOnly, job.SequenceConfig, sequenceMap)
	}

	return CharCountResult{
		CharMap:     charMap,
		SequenceMap: sequenceMap,
		FileCount:   1,
		CharCount:   charCount,
	}
}

func extractSequences(content string, asciiOnly bool, config SequenceConfig, sequenceMap map[string]int) {
	runes := []rune(strings.ToLower(content))

	var cleanRunes []rune
	for _, r := range runes {
		if !unicode.IsSpace(r) {
			if asciiOnly && r > 127 {
				continue
			}
			if unicode.IsGraphic(r) || unicode.IsControl(r) {
				cleanRunes = append(cleanRunes, r)
			}
		}
	}

	for length := config.MinLength; length <= config.MaxLength; length++ {
		for i := 0; i <= len(cleanRunes)-length; i++ {
			seq := string(cleanRunes[i : i+length])
			sequenceMap[seq]++
		}
	}
}
