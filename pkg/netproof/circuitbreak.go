package netproof

import (
	"errors"
	"time"
)

var ErrCircuitBreak = errors.New("circuit break")

type CircuitBreak struct {
	remaining       int
	maxFailures     int
	lastFailureTime time.Time
	resetTimeout    time.Duration
	halfOpen        bool
	t               timer
}

// NewCircuitBreak creates a new circuit breaker.
func NewCircuitBreak(maxFailures int, resetTimeout time.Duration) *CircuitBreak {
	return newCircuitBreak(maxFailures, resetTimeout, realTimer{})
}

func newCircuitBreak(maxFailures int, resetTimeout time.Duration, t timer) *CircuitBreak {
	return &CircuitBreak{
		maxFailures:  maxFailures,
		remaining:    maxFailures,
		resetTimeout: resetTimeout,
		t:            t,
	}
}

// CircuitBreak runs the given function f().
// After (consecutive) maxFailures it'll return ErrCircuitBreak
// It'll reset after resetTimeout (in half-open state).
func (cb *CircuitBreak) CircuitBreak(f func() error) error {
	if cb.remaining == 0 {
		if cb.t.Since(cb.lastFailureTime) < cb.resetTimeout {
			return ErrCircuitBreak
		}
		cb.halfOpen = true
	}
	err := f()
	if err != nil {
		cb.remaining--
		cb.lastFailureTime = cb.t.Now()
		if cb.halfOpen {
			cb.remaining = 0
		}
		return err
	}
	cb.remaining = cb.maxFailures
	cb.halfOpen = false
	return nil
}
