package main

import (
	"sync"
	"time"
)

type Clock interface{ Now() time.Time }
type Limiter struct {
	mu     sync.Mutex
	clock  Clock
	ratePerSec float64
	burst  int
	currToken float64
	lastAdd time.Time
}

func NewLimiter(clock Clock, ratePerSec float64, burst int) *Limiter {
	var lastAdd time.Time

	if clock != nil {
		lastAdd = clock.Now()
	}
	
	return &Limiter{
		clock: clock,
		ratePerSec: ratePerSec,
		burst: burst,
		currToken: float64(burst),
		lastAdd: lastAdd, 
	}
}

func (l *Limiter) AllowN(n int) bool {
	if n <= 0 || n > l.burst || l.burst <= 0 || l.clock == nil {
		return false
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.ratePerSec > 0 {
		currTime := l.clock.Now()
		diff := currTime.Sub(l.lastAdd).Seconds()

		if diff > 0 {	
			l.lastAdd = currTime
			addToken := l.ratePerSec * diff
			l.currToken += addToken	
		}
	}
	
	if l.currToken > float64(l.burst) {
		l.currToken = float64(l.burst)
	}

	if float64(n) > l.currToken{
		return false
	}
	l.currToken -= float64(n)
	return true
}