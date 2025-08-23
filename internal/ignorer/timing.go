package ignorer

import (
	"sync/atomic"
	"time"

	"github.com/ogdakke/symbolista/internal/logger"
)

type TimingMatcher struct {
	*Matcher
	loadTime  int64 // nanoseconds, atomic
	matchTime int64 // nanoseconds, atomic
}

func NewTimingMatcher(basePath string, includeDotfiles bool) (*TimingMatcher, error) {
	loadStart := time.Now()
	matcher, err := NewMatcher(basePath, includeDotfiles)
	loadDuration := time.Since(loadStart)

	if err != nil {
		return nil, err
	}

	tm := &TimingMatcher{
		Matcher: matcher,
	}
	atomic.AddInt64(&tm.loadTime, int64(loadDuration))

	logger.Debug("Timing matcher created", "initial_load_duration", loadDuration)
	return tm, nil
}

func (tm *TimingMatcher) LoadGitignoreForDirectory(dirPath string) error {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		atomic.AddInt64(&tm.loadTime, int64(duration))
		logger.Trace("Gitignore load timing", "dir", dirPath, "duration", duration)
	}()

	return tm.Matcher.LoadGitignoreForDirectory(dirPath)
}

func (tm *TimingMatcher) ShouldIgnore(path string) bool {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		atomic.AddInt64(&tm.matchTime, int64(duration))
		if duration > time.Microsecond*100 {
			logger.Trace("Gitignore match timing", "path", path, "duration", duration)
		}
	}()

	return tm.Matcher.ShouldIgnore(path)
}

func (tm *TimingMatcher) GetLoadTime() time.Duration {
	if tm == nil {
		return 0
	}
	return time.Duration(atomic.LoadInt64(&tm.loadTime))
}

func (tm *TimingMatcher) GetMatchTime() time.Duration {
	if tm == nil {
		return 0
	}
	return time.Duration(atomic.LoadInt64(&tm.matchTime))
}

func (tm *TimingMatcher) GetTotalTime() time.Duration {
	if tm == nil {
		return 0
	}
	return tm.GetLoadTime() + tm.GetMatchTime()
}
