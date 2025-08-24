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
	charCount := 0

	logger.Trace("Processing file", "path", job.Path, "worker_id", workerID, "size", len(job.Content))

	content := strings.ToLower(string(job.Content))
	n := len(content)

	for _, r := range content {
		if !unicode.IsGraphic(r) && !unicode.IsSpace(r) {
			continue
		}

		if job.AsciiOnly && r > 127 {
			continue
		}

		charMap[r]++
		charCount++
	}

	sequenceMap2 := make(map[uint16]uint32, n)
	sequenceMap3 := make(map[uint32]uint32, n)
	if job.SequenceConfig.Enabled {

		var b0, b1, b2 uint32

		for i := range n {
			b2 = uint32(content[i])

			if i >= 1 {
				k2 := uint16((b1 << 8) | b2)
				sequenceMap2[k2]++
			}
			if i >= 2 {
				k3 := (b0 << 16) | (b1 << 8) | b2
				sequenceMap3[k3]++
			}

			b0, b1 = b1, b2
		}
	}
	return CharCountResult{
		CharMap:      charMap,
		SequenceMap2: sequenceMap2,
		SequenceMap3: sequenceMap3,
		FileCount:    1,
		CharCount:    charCount,
	}
}
