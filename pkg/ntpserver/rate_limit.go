package ntpserver

import (
	"sync"
	"time"
)

type tokenBucket struct {
	mu       sync.Mutex
	tokens   float64
	last     time.Time
	rate     float64
	burst    float64
}

func newTokenBucket(ratePerSec float64, burst int) *tokenBucket {
	if ratePerSec <= 0 {
		ratePerSec = 0
	}
	if burst <= 0 {
		burst = 1
	}
	return &tokenBucket{
		tokens: float64(burst),
		last:  time.Now(),
		rate:  ratePerSec,
		burst: float64(burst),
	}
}

func (b *tokenBucket) allow(now time.Time) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.rate <= 0 {
		return true
	}

	dt := now.Sub(b.last).Seconds()
	if dt < 0 {
		dt = 0
	}
	b.last = now
	b.tokens += dt * b.rate
	if b.tokens > b.burst {
		b.tokens = b.burst
	}
	if b.tokens < 1 {
		return false
	}
	b.tokens -= 1
	return true
}

type limiter struct {
	mu    sync.Mutex
	perIP map[string]*tokenBucket

	ratePerSec float64
	burst      int
}

func newLimiter(ratePerSec float64, burst int) *limiter {
	return &limiter{
		perIP:       make(map[string]*tokenBucket),
		ratePerSec:  ratePerSec,
		burst:       burst,
	}
}

func (l *limiter) allow(ip string, now time.Time) bool {
	if l.ratePerSec <= 0 {
		return true
	}
	if ip == "" {
		return true
	}

	l.mu.Lock()
	b := l.perIP[ip]
	if b == nil {
		b = newTokenBucket(l.ratePerSec, l.burst)
		l.perIP[ip] = b
	}
	l.mu.Unlock()

	return b.allow(now)
}
