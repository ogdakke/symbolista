package concurrent

import (
	"runtime"
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

	// Start a goroutine to close results channel when all workers are done
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

	for _, r := range string(job.Content) {
		if unicode.IsGraphic(r) || unicode.IsSpace(r) {
			charMap[r]++
			charCount++
		}
	}

	return CharCountResult{
		CharMap:   charMap,
		FileCount: 1,
		CharCount: charCount,
	}
}
