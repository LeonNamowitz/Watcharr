package tmdb

import (
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const maxRateLimitRetries = 2

// Share pacing across stats, search and detail calls, including TMDB instances.
// Twenty requests per second leaves room below TMDB's approximate 40/s ceiling.
var tmdbRequests = &requestPacer{interval: 50 * time.Millisecond}

type requestPacer struct {
	mu       sync.Mutex
	interval time.Duration
	next     time.Time
}

func (p *requestPacer) wait() {
	for {
		p.mu.Lock()
		delay := time.Until(p.next)
		if delay <= 0 {
			p.next = time.Now().Add(p.interval)
			p.mu.Unlock()
			return
		}
		p.mu.Unlock()
		time.Sleep(delay)
		// Recheck after sleeping so a 429 pauses callers already in the queue.
	}
}

func (p *requestPacer) backoff(delay time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if until := time.Now().Add(delay); until.After(p.next) {
		p.next = until
	}
}

func rateLimitDelay(value string, attempt int, now time.Time) time.Duration {
	value = strings.TrimSpace(value)
	if seconds, err := strconv.ParseInt(value, 10, 32); err == nil && seconds >= 0 {
		return time.Duration(seconds) * time.Second
	}
	if until, err := http.ParseTime(value); err == nil {
		return max(until.Sub(now), 0)
	}
	// Retry-After is optional; otherwise use bounded exponential backoff.
	return time.Second << attempt
}
